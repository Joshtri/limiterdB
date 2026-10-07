package domain

import "context"

type LevelMeter interface {
	Read(ctx context.Context) (Level, error)
}

type VolumeController interface {
	Get() (Volume, error)
	Set(v Volume) error
}

// DeviceSource memberi info output device default saat ini.
type DeviceSource interface {
	Device() (Device, error)
}

// ExposureStore menyimpan catatan paparan ke penyimpanan permanen.
type ExposureStore interface {
	LoadExposure() (ExposureLog, error)
	SaveExposure(ExposureLog) error
}

// SettingsStore menyimpan pengaturan user ke penyimpanan permanen.
type SettingsStore interface {
	LoadSettings() (Settings, error)
	SaveSettings(Settings) error
}
