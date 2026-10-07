//go:build windows

package adapter

import (
	"math"

	"limiterdB/internal/domain"
)

// Volume di domain adalah gain linear (amplitudo 0..1). Slider/scalar
// Windows (GetMasterVolumeLevelScalar) memakai kurva taper yang berbeda per
// device, jadi di sini volume dibaca/ditulis dalam dB supaya
// 20·log10(gain) benar-benar sama dengan redaman yang diterapkan.

// Get mengembalikan gain endpoint render default saat ini (0..1).
// Mengimplementasikan domain.VolumeController.
func (a *Adapter) Get() (domain.Volume, error) {
	var db float32
	if err := a.exec(func(e *endpoint) error {
		return e.volume.GetMasterVolumeLevel(&db)
	}); err != nil {
		return 0, err
	}
	return domain.VolumeFromDB(float64(db)), nil
}

// Set mengatur gain endpoint render default (0..1), dibatasi ke rentang dB
// yang didukung device. Mengimplementasikan domain.VolumeController.
func (a *Adapter) Set(v domain.Volume) error {
	return a.exec(func(e *endpoint) error {
		db := e.info.MinDB
		if v > 0 {
			db = math.Max(e.info.MinDB, math.Min(e.info.MaxDB, 20*math.Log10(float64(v))))
		}
		return e.volume.SetMasterVolumeLevel(float32(db), nil)
	})
}
