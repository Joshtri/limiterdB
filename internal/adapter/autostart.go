//go:build windows

package adapter

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Auto-start lewat HKCU\...\Run: tidak butuh hak admin, berlaku untuk user
// yang sedang login saja.
const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "limiterdB"
	// MinimizedFlag dipakai saat dijalankan otomatis: langsung ke tray.
	MinimizedFlag = "--minimized"
)

// AutoStartEnabled mengecek apakah limiterdB terdaftar untuk jalan saat login.
func AutoStartEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()

	v, _, err := k.GetStringValue(runValueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(v) != "", nil
}

// SetAutoStart mendaftarkan/menghapus exe ini dari daftar startup Windows.
func SetAutoStart(on bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !on {
		if err := k.DeleteValue(runValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Exe dari `wails dev` butuh dev server yang jalan → tidak berguna saat login.
	if strings.HasSuffix(strings.ToLower(exe), "-dev.exe") {
		return errors.New("auto-start hanya bisa diaktifkan dari hasil `wails build` (build\\bin\\limiterdB.exe), bukan dari `wails dev`")
	}
	return k.SetStringValue(runValueName, fmt.Sprintf(`"%s" %s`, exe, MinimizedFlag))
}
