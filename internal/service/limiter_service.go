package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"limiterdB/internal/domain"
)

var (
	ErrAlreadyRunning = errors.New("limiter sudah berjalan")
	ErrNoDevice       = errors.New("tidak ada perangkat output aktif")
)

const (
	// deviceRefreshTicks: info device (nama, mute) dibaca ulang tiap N tick.
	deviceRefreshTicks = 10
	// exposureSaveEvery: seberapa sering catatan paparan ditulis ke disk.
	exposureSaveEvery = time.Minute
	// exposureKeepDays: catatan lebih lama dari ini dibuang.
	exposureKeepDays = 30
)

// Status adalah snapshot read-only buat ditampilkan ke UI.
type Status struct {
	Enabled       bool          `json:"enabled"`
	CurrentVolume float64       `json:"currentVolume"`
	LastLevel     float64       `json:"lastLevel"`
	Threshold     float64       `json:"threshold"`
	Ceiling       float64       `json:"ceiling"`
	TargetSPL     float64       `json:"targetSpl"`
	MaxSPL        float64       `json:"maxSpl"`
	Calibrated    bool          `json:"calibrated"`
	HeardSPL      float64       `json:"heardSpl"`  // perkiraan dB SPL saat ini
	Listening     bool          `json:"listening"` // ada audio terdengar (tidak senyap/mute)
	DeviceOK      bool          `json:"deviceOk"`
	Device        domain.Device `json:"device"`
}

// LimiterService menjalankan loop yang selalu aktif selama aplikasi hidup:
// baca level & volume → catat paparan → (kalau limiter aktif) hitung dan
// terapkan volume ideal lewat domain.Limiter.
type LimiterService struct {
	limiter  *domain.Limiter
	meter    domain.LevelMeter
	volume   domain.VolumeController
	devices  domain.DeviceSource
	store    domain.ExposureStore
	interval time.Duration
	now      func() time.Time

	mu       sync.RWMutex
	settings domain.Settings
	running  bool
	cancel   context.CancelFunc
	loopDone chan struct{}

	device    domain.Device
	deviceOK  bool
	lastLevel domain.Level
	lastVol   domain.Volume
	exposure  domain.ExposureLog
}

func NewLimiterService(
	limiter *domain.Limiter,
	meter domain.LevelMeter,
	volume domain.VolumeController,
	devices domain.DeviceSource,
	store domain.ExposureStore,
	settings domain.Settings,
	exposure domain.ExposureLog,
	interval time.Duration,
) *LimiterService {
	if exposure.Days == nil {
		exposure = domain.NewExposureLog()
	}
	return &LimiterService{
		limiter:   limiter,
		meter:     meter,
		volume:    volume,
		devices:   devices,
		store:     store,
		interval:  interval,
		now:       time.Now,
		settings:  settings.Normalize(),
		exposure:  exposure,
		lastLevel: domain.Level(-100),
	}
}

// Start menjalankan loop di goroutine baru. Tidak blocking.
func (s *LimiterService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.running = true
	done := make(chan struct{})
	s.loopDone = done
	s.mu.Unlock()

	go s.loop(loopCtx, done)
	return nil
}

// Stop membatalkan loop, menunggu sampai benar-benar berhenti, lalu
// menyimpan catatan paparan terakhir.
func (s *LimiterService) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	done := s.loopDone
	s.cancel = nil
	s.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	<-done
	s.saveExposure()
}

func (s *LimiterService) loop(ctx context.Context, done chan struct{}) {
	defer close(done)
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.refreshDevice()
	last := s.now()
	lastSave := last
	n := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := s.now()
			// Kalau tick telat jauh (laptop sleep), jangan dihitung sebagai waktu dengar.
			dt := now.Sub(last)
			if dt > 4*s.interval {
				dt = s.interval
			}
			last = now

			n++
			if n%deviceRefreshTicks == 0 {
				s.refreshDevice()
			}
			s.tick(ctx, now, dt)

			if now.Sub(lastSave) >= exposureSaveEvery {
				lastSave = now
				s.saveExposure()
			}
		}
	}
}

func (s *LimiterService) refreshDevice() {
	d, err := s.devices.Device()
	s.mu.Lock()
	s.deviceOK = err == nil
	if err == nil {
		s.device = d
	}
	s.mu.Unlock()
}

func (s *LimiterService) tick(ctx context.Context, now time.Time, dt time.Duration) {
	lvl, err := s.meter.Read(ctx)
	if err != nil {
		return
	}
	cur, err := s.volume.Get()
	if err != nil {
		return
	}

	s.mu.Lock()
	st := s.settings
	maxSPL, _ := st.MaxSPLFor(s.device)
	th := domain.ThresholdFor(st.TargetSPL, maxSPL)

	next := cur
	if st.LimiterEnabled {
		next = s.limiter.NextVolume(cur, lvl, th, domain.NewVolume(st.Ceiling))
	}

	// Yang terdengar selama tick ini memakai volume yang sedang berlaku (cur).
	if s.deviceOK && !s.device.Muted && lvl > domain.SilenceLevel {
		s.exposure.Add(now, dt, domain.EstimateSPL(maxSPL, lvl, cur))
	}

	s.lastLevel = lvl
	s.lastVol = next
	s.mu.Unlock()

	if next != cur {
		_ = s.volume.Set(next)
	}
}

func (s *LimiterService) saveExposure() {
	if s.store == nil {
		return
	}
	s.mu.Lock()
	s.exposure.Prune(s.now(), exposureKeepDays)
	snapshot := s.exposure.Clone()
	s.mu.Unlock()
	_ = s.store.SaveExposure(snapshot)
}

// --- pengaturan (aman dipanggil saat loop berjalan) ---

func (s *LimiterService) update(fn func(*domain.Settings)) domain.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.settings)
	s.settings = s.settings.Normalize()
	return s.settings
}

// SetEnabled menyalakan/mematikan penyesuaian volume. Monitoring & catatan
// paparan tetap berjalan.
func (s *LimiterService) SetEnabled(on bool) domain.Settings {
	return s.update(func(st *domain.Settings) { st.LimiterEnabled = on })
}

func (s *LimiterService) SetTargetSPL(spl float64) domain.Settings {
	return s.update(func(st *domain.Settings) { st.TargetSPL = spl })
}

func (s *LimiterService) SetCeiling(v float64) domain.Settings {
	return s.update(func(st *domain.Settings) { st.Ceiling = v })
}

// SetCalibration menyimpan maxSPL untuk device yang sedang aktif.
func (s *LimiterService) SetCalibration(maxSPL float64) (domain.Settings, error) {
	s.mu.RLock()
	id, ok := s.device.ID, s.deviceOK
	s.mu.RUnlock()
	if !ok || id == "" {
		return s.Settings(), ErrNoDevice
	}
	return s.update(func(st *domain.Settings) { st.Calibrations[id] = maxSPL }), nil
}

// ResetCalibration menghapus kalibrasi device yang sedang aktif.
func (s *LimiterService) ResetCalibration() domain.Settings {
	s.mu.RLock()
	id := s.device.ID
	s.mu.RUnlock()
	return s.update(func(st *domain.Settings) { delete(st.Calibrations, id) })
}

// Settings mengembalikan salinan pengaturan saat ini.
func (s *LimiterService) Settings() domain.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.settings
	c.Calibrations = make(map[string]float64, len(s.settings.Calibrations))
	for k, v := range s.settings.Calibrations {
		c.Calibrations[k] = v
	}
	return c
}

// Exposure mengembalikan ringkasan paparan 7 hari terakhir.
func (s *LimiterService) Exposure() domain.ExposureSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.exposure.Summary(s.now())
}

// Status mengembalikan snapshot dari tick terakhir.
func (s *LimiterService) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	maxSPL, calibrated := s.settings.MaxSPLFor(s.device)
	listening := s.deviceOK && !s.device.Muted && s.lastLevel > domain.SilenceLevel
	return Status{
		Enabled:       s.settings.LimiterEnabled,
		CurrentVolume: float64(s.lastVol),
		LastLevel:     float64(s.lastLevel),
		Threshold:     float64(domain.ThresholdFor(s.settings.TargetSPL, maxSPL)),
		Ceiling:       s.settings.Ceiling,
		TargetSPL:     s.settings.TargetSPL,
		MaxSPL:        maxSPL,
		Calibrated:    calibrated,
		HeardSPL:      domain.EstimateSPL(maxSPL, s.lastLevel, s.lastVol),
		Listening:     listening,
		DeviceOK:      s.deviceOK,
		Device:        s.device,
	}
}
