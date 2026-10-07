// Package store menyimpan pengaturan dan catatan paparan sebagai file JSON
// di folder konfigurasi user (Windows: %AppData%\limiterdB).
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"limiterdB/internal/domain"
)

const (
	settingsFile = "settings.json"
	exposureFile = "exposure.json"
)

// Store mengimplementasikan domain.SettingsStore dan domain.ExposureStore.
type Store struct {
	dir string
	mu  sync.Mutex // satu penulis per waktu
}

// New membuat Store di dir (dibuat kalau belum ada).
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// DefaultDir mengembalikan %AppData%\limiterdB.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "limiterdB"), nil
}

// Dir mengembalikan folder penyimpanan.
func (s *Store) Dir() string { return s.dir }

// LoadSettings membaca pengaturan; file belum ada → pengaturan default.
func (s *Store) LoadSettings() (domain.Settings, error) {
	st := domain.DefaultSettings()
	if err := s.read(settingsFile, &st); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.DefaultSettings(), nil
		}
		return domain.DefaultSettings(), err
	}
	return st.Normalize(), nil
}

func (s *Store) SaveSettings(st domain.Settings) error { return s.write(settingsFile, st) }

// LoadExposure membaca catatan paparan; file belum ada → catatan kosong.
func (s *Store) LoadExposure() (domain.ExposureLog, error) {
	l := domain.NewExposureLog()
	if err := s.read(exposureFile, &l); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.NewExposureLog(), nil
		}
		return domain.NewExposureLog(), err
	}
	if l.Days == nil {
		l = domain.NewExposureLog()
	}
	return l, nil
}

func (s *Store) SaveExposure(l domain.ExposureLog) error { return s.write(exposureFile, l) }

func (s *Store) read(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// write menulis ke file sementara lalu rename, supaya file tidak pernah
// setengah tertulis kalau aplikasi mati di tengah jalan.
func (s *Store) write(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
