//go:build windows

package adapter

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"

	"limiterdB/internal/domain"
)

// deviceCheckInterval: seberapa sering default output device dicek ulang.
// Kalau user ganti device (colok headset, pilih speaker lain di Windows),
// adapter pindah otomatis paling lambat selang ini.
const deviceCheckInterval = 500 * time.Millisecond

var (
	// ErrClosed dikembalikan exec setelah Adapter ditutup.
	ErrClosed = errors.New("wasapi: adapter sudah ditutup")
	// ErrNoDevice dikembalikan saat tidak ada output device aktif.
	ErrNoDevice = errors.New("wasapi: tidak ada perangkat output aktif")
)

// endpoint memegang handle COM untuk satu output device.
type endpoint struct {
	info   domain.Device
	device *wca.IMMDevice
	volume *wca.IAudioEndpointVolume
	meter  *wca.IAudioMeterInformation
}

// openEndpoint mengaktifkan volume & meter dari device dan membaca
// propertinya. Kepemilikan device pindah ke endpoint (dirilis lewat
// release), juga saat gagal.
func openEndpoint(device *wca.IMMDevice, id string) (*endpoint, error) {
	e := &endpoint{device: device, info: domain.Device{ID: id}}

	if err := device.Activate(wca.IID_IAudioEndpointVolume, wca.CLSCTX_ALL, nil, &e.volume); err != nil {
		e.release()
		return nil, fmt.Errorf("wasapi: Activate(IAudioEndpointVolume): %w", err)
	}
	if err := device.Activate(wca.IID_IAudioMeterInformation, wca.CLSCTX_ALL, nil, &e.meter); err != nil {
		e.release()
		return nil, fmt.Errorf("wasapi: Activate(IAudioMeterInformation): %w", err)
	}

	var minDB, maxDB, incDB float32
	if err := e.volume.GetVolumeRange(&minDB, &maxDB, &incDB); err != nil {
		e.release()
		return nil, fmt.Errorf("wasapi: GetVolumeRange: %w", err)
	}
	e.info.MinDB, e.info.MaxDB = float64(minDB), float64(maxDB)

	// Nama & jenis device cuma buat ditampilkan; gagal baca bukan error fatal.
	e.info.Name, e.info.Kind = readDeviceProps(device)
	return e, nil
}

func readDeviceProps(device *wca.IMMDevice) (name, kind string) {
	kind = "other"
	var ps *wca.IPropertyStore
	if err := device.OpenPropertyStore(wca.STGM_READ, &ps); err != nil {
		return "", kind
	}
	defer ps.Release()

	var pv wca.PROPVARIANT
	if err := ps.GetValue(&wca.PKEY_Device_FriendlyName, &pv); err == nil && pv.VT == ole.VT_LPWSTR {
		name = pv.String() // String() juga membebaskan buffer-nya
	}

	var ff wca.PROPVARIANT
	if err := ps.GetValue(&wca.PKEY_AudioEndpoint_FormFactor, &ff); err == nil && ff.VT == ole.VT_UI4 {
		kind = formFactorKind(uint32(ff.Val))
	}
	return name, kind
}

// formFactorKind memetakan EndpointFormFactor Windows ke kategori sederhana.
func formFactorKind(ff uint32) string {
	switch ff {
	case 3: // Headphones
		return "headphones"
	case 5, 6: // Headset, Handset
		return "headset"
	case 1: // Speakers
		return "speakers"
	case 9: // DigitalAudioDisplayDevice (HDMI/DisplayPort)
		return "display"
	}
	return "other"
}

// muted membaca status mute. Tidak memakai wca.GetMute karena wrapper itu
// menulis BOOL Windows (4 byte) ke *bool Go (1 byte).
func (e *endpoint) muted() (bool, error) {
	var b int32
	hr, _, _ := syscall.SyscallN(
		e.volume.VTable().GetMute,
		uintptr(unsafe.Pointer(e.volume)),
		uintptr(unsafe.Pointer(&b)),
	)
	if hr != 0 {
		return false, ole.NewError(hr)
	}
	return b != 0, nil
}

func (e *endpoint) release() {
	if e.meter != nil {
		e.meter.Release()
	}
	if e.volume != nil {
		e.volume.Release()
	}
	if e.device != nil {
		e.device.Release()
	}
}

// session memegang handle COM/WASAPI ke default audio render endpoint.
//
// COM object yang dibuat lewat CoInitializeEx terikat ke thread yang
// menginisialisasinya (apartment-threaded) — goroutine Go bisa pindah OS
// thread kapan saja, jadi session ini WAJIB dibuat dan dipakai dari satu
// goroutine yang sudah di-runtime.LockOSThread(), dan ditutup dari goroutine
// yang sama.
type session struct {
	enumerator *wca.IMMDeviceEnumerator
	ep         *endpoint // nil kalau tidak ada output device aktif
	stale      bool      // command terakhir gagal → buka ulang device walau ID sama
}

// newSession menginisialisasi COM dan membuat device enumerator. Tidak
// adanya output device bukan error: session tetap jalan dan menunggu device
// muncul lewat refresh.
func newSession() (*session, error) {
	runtime.LockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("wasapi: CoInitializeEx: %w", err)
	}

	s := &session{}
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&s.enumerator,
	); err != nil {
		s.close()
		return nil, fmt.Errorf("wasapi: CoCreateInstance(MMDeviceEnumerator): %w", err)
	}

	_ = s.refresh()
	return s, nil
}

// refresh memastikan session memegang default output device saat ini. Kalau
// default device berganti (atau command sebelumnya gagal), endpoint lama
// dilepas dan yang baru dibuka.
func (s *session) refresh() error {
	var device *wca.IMMDevice
	if err := s.enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &device); err != nil {
		s.drop()
		return ErrNoDevice
	}

	var id string
	if err := device.GetId(&id); err != nil {
		device.Release()
		s.drop()
		return fmt.Errorf("wasapi: GetId: %w", err)
	}

	if s.ep != nil && s.ep.info.ID == id && !s.stale {
		device.Release()
		return nil
	}

	ep, err := openEndpoint(device, id)
	s.drop()
	if err != nil {
		return err
	}
	s.ep = ep
	return nil
}

func (s *session) drop() {
	if s.ep != nil {
		s.ep.release()
		s.ep = nil
	}
	s.stale = false
}

// close melepas semua COM object dan mengembalikan thread ini ke pool
// scheduler Go. Harus dipanggil dari goroutine yang sama dengan newSession().
func (s *session) close() {
	s.drop()
	if s.enumerator != nil {
		s.enumerator.Release()
	}
	ole.CoUninitialize()
	runtime.UnlockOSThread()
}

// Adapter menjalankan satu goroutine dedicated yang memegang session COM dan
// mengeksekusi semua command WASAPI secara serial di goroutine itu. Metode
// domain.LevelMeter dan domain.VolumeController ada di meter.go dan
// volume.go, keduanya metode pada Adapter ini supaya berbagi session yang
// sama tanpa melanggar aturan threading COM.
type Adapter struct {
	cmds chan func(*session)
	done chan struct{}

	// mu menjaga closed: exec memegang RLock selama mengirim command,
	// Close memegang Lock sebelum menutup cmds, jadi exec tidak pernah
	// mengirim ke channel yang sudah ditutup.
	mu     sync.RWMutex
	closed bool
}

// NewAdapter membuat goroutine dedicated, menginisialisasi session WASAPI di
// dalamnya, dan menunggu sampai siap (atau gagal) sebelum return.
func NewAdapter() (*Adapter, error) {
	a := &Adapter{
		cmds: make(chan func(*session)),
		done: make(chan struct{}),
	}

	ready := make(chan error, 1)
	go a.run(ready)

	if err := <-ready; err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Adapter) run(ready chan error) {
	s, err := newSession()
	ready <- err
	if err != nil {
		close(a.done)
		return
	}
	defer s.close()
	defer close(a.done)

	ticker := time.NewTicker(deviceCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case cmd, ok := <-a.cmds:
			if !ok {
				return
			}
			cmd(s)
		case <-ticker.C:
			_ = s.refresh()
		}
	}
}

// exec menjalankan fn terhadap endpoint aktif di goroutine dedicated milik
// Adapter dan menunggu hasilnya. Kalau fn gagal (misal device dicabut),
// endpoint ditandai stale supaya dibuka ulang pada pengecekan berikutnya.
func (a *Adapter) exec(fn func(*endpoint) error) error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.closed {
		return ErrClosed
	}

	resultErr := make(chan error, 1)
	a.cmds <- func(s *session) {
		if s.ep == nil {
			if err := s.refresh(); err != nil {
				resultErr <- err
				return
			}
		}
		err := fn(s.ep)
		if err != nil {
			s.stale = true
		}
		resultErr <- err
	}
	return <-resultErr
}

// Device mengembalikan info output device default yang sedang dipakai.
// Mengimplementasikan domain.DeviceSource.
func (a *Adapter) Device() (domain.Device, error) {
	var info domain.Device
	err := a.exec(func(e *endpoint) error {
		muted, err := e.muted()
		if err != nil {
			return err
		}
		info = e.info
		info.Muted = muted
		return nil
	})
	return info, err
}

// Close menghentikan goroutine dedicated dan melepas session COM-nya.
// Sebaiknya dipanggil setelah LimiterService berhenti; exec yang masih
// datang sesudahnya (misal polling Status dari UI saat window ditutup)
// mendapat ErrClosed, bukan panic.
func (a *Adapter) Close() {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true
	close(a.cmds)
	a.mu.Unlock()
	<-a.done
}
