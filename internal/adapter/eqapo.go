//go:build windows

package adapter

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Boost volume di atas 100% lewat Equalizer APO (https://sourceforge.net/projects/equalizerapo/).
// Windows membatasi volume endpoint di 0 dB, jadi gain tambahan harus dipasang
// di dalam pipeline audio — itulah fungsi Equalizer APO.
//
// Cara kerjanya:
//   - Sekali saja ("hubungkan"): tambahkan `Include: limiterdB-boost.txt` di
//     akhir config.txt Equalizer APO, buat file boost di folder config, dan
//     beri grup Users izin menulis file itu. Folder config ada di Program Files,
//     jadi langkah ini biasanya butuh admin (UAC).
//   - Selanjutnya limiterdB cukup menulis ulang file boost (`Preamp: +X dB`)
//     tanpa admin. Equalizer APO memuat ulang otomatis kalau file di folder
//     config berubah.
const (
	eqapoBoostFile = "limiterdB-boost.txt"
	eqapoInclude   = "Include: " + eqapoBoostFile
	// EqAPOLinkFlag: exe ini dijalankan ulang sebagai admin dengan flag ini
	// untuk menghubungkan Equalizer APO, lalu langsung keluar.
	EqAPOLinkFlag = "--eqapo-link"
	// usersSID = BUILTIN\Users (tidak tergantung bahasa Windows).
	usersSID = "*S-1-5-32-545"
)

// ErrEqAPONotLinked: Equalizer APO terpasang tapi belum dihubungkan (atau
// baris Include-nya terhapus, misal config.txt ditimpa Peace).
var ErrEqAPONotLinked = errors.New("Equalizer APO belum dihubungkan ke limiterdB")

// EqualizerAPO menulis boost ke konfigurasi Equalizer APO.
type EqualizerAPO struct {
	configDir string
}

// FindEqualizerAPO mencari instalasi Equalizer APO lewat registry.
func FindEqualizerAPO() (*EqualizerAPO, bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\EqualizerAPO`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return nil, false
	}
	defer k.Close()
	dir, _, err := k.GetStringValue("ConfigPath")
	if err != nil || strings.TrimSpace(dir) == "" {
		return nil, false
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, false
	}
	return &EqualizerAPO{configDir: dir}, true
}

func (e *EqualizerAPO) ConfigDir() string { return e.configDir }

func (e *EqualizerAPO) configFile() string { return filepath.Join(e.configDir, "config.txt") }
func (e *EqualizerAPO) boostFile() string  { return filepath.Join(e.configDir, eqapoBoostFile) }

// Linked mengecek apakah config.txt memuat file boost limiterdB.
func (e *EqualizerAPO) Linked() bool {
	data, err := os.ReadFile(e.configFile())
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.EqualFold(strings.TrimSpace(line), eqapoInclude) {
			return true
		}
	}
	return false
}

// SetBoost menulis gain tambahan (dB, ≥ 0). Mengimplementasikan domain.Booster.
// Boost 0 saat belum terhubung bukan error (memang tidak ada yang perlu diubah).
func (e *EqualizerAPO) SetBoost(db float64) error {
	if !e.Linked() {
		if db == 0 {
			return nil
		}
		return ErrEqAPONotLinked
	}
	return os.WriteFile(e.boostFile(), boostConfig(db), 0o644)
}

func boostConfig(db float64) []byte {
	return fmt.Appendf(nil, "# Dikelola oleh limiterdB (boost speaker) — jangan diedit manual.\r\nPreamp: %.2f dB\r\n", db)
}

// Link menghubungkan Equalizer APO ke limiterdB. Butuh izin menulis ke folder
// config; kalau ditolak, error-nya fs.ErrPermission (pakai LinkElevated).
func (e *EqualizerAPO) Link() error {
	if _, err := os.Stat(e.boostFile()); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(e.boostFile(), boostConfig(0), 0o644); err != nil {
			return err
		}
	}
	// Supaya limiterdB (tanpa admin) bisa mengubah boost setelah ini.
	// Gagal di sini tidak fatal kalau folder memang sudah bisa ditulis user.
	icacls := exec.Command("icacls", e.boostFile(), "/grant", usersSID+":M")
	icacls.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = icacls.Run()

	if e.Linked() {
		return nil
	}
	data, err := os.ReadFile(e.configFile())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// Di akhir file supaya boost berlaku setelah filter lain milik user.
	var buf bytes.Buffer
	buf.Write(data)
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		buf.WriteString("\r\n")
	}
	buf.WriteString("# limiterdB: boost volume speaker\r\n" + eqapoInclude + "\r\n")
	return os.WriteFile(e.configFile(), buf.Bytes(), 0o644)
}

// LinkElevated menjalankan exe ini sebagai admin (muncul prompt UAC) untuk
// menjalankan Link. Tidak menunggu selesai; cek Linked() setelahnya.
func LinkElevated() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exe)
	args, _ := windows.UTF16PtrFromString(EqAPOLinkFlag)
	err = windows.ShellExecute(0, verb, file, args, nil, windows.SW_HIDE)
	if errors.Is(err, windows.ERROR_CANCELLED) {
		return errors.New("izin admin dibatalkan")
	}
	return err
}

// RunLinkHelper dipanggil dari main saat exe dijalankan dengan EqAPOLinkFlag.
func RunLinkHelper() int {
	eq, ok := FindEqualizerAPO()
	if !ok {
		return 2
	}
	if err := eq.Link(); err != nil {
		return 1
	}
	return 0
}
