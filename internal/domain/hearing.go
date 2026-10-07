package domain

import "math"

// Perkiraan dB SPL (tekanan suara di telinga):
//
//	dB SPL ≈ maxSPL + level (dBFS) + gain volume (dB)
//
// maxSPL = seberapa keras device di volume 100% dengan sinyal 0 dBFS. Windows
// tidak tahu nilai ini, jadi dipakai default per jenis device atau hasil
// kalibrasi user.

// SilenceLevel: level di bawah ini dianggap tidak ada audio yang diputar.
const SilenceLevel Level = -59

var defaultMaxSPL = map[string]float64{
	"headphones": 100,
	"headset":    100,
	"speakers":   85,
	"display":    85,
	"other":      95,
}

// DefaultMaxSPL mengembalikan perkiraan maxSPL untuk jenis device.
func DefaultMaxSPL(kind string) float64 {
	if v, ok := defaultMaxSPL[kind]; ok {
		return v
	}
	return defaultMaxSPL["other"]
}

// ThresholdFor menghitung threshold dBFS supaya yang terdengar tidak melewati
// targetSPL, dibatasi ke rentang threshold yang valid.
func ThresholdFor(targetSPL, maxSPL float64) Threshold {
	return Threshold(math.Max(-60, math.Min(0, math.Round(targetSPL-maxSPL))))
}

const (
	// SafeSPL: WHO, rata-rata 80 dB aman ±40 jam/minggu.
	SafeSPL = 80.0
	// LimitSPL: NIOSH, 85 dB maksimal 8 jam/hari.
	LimitSPL = 85.0
)

// ZoneOf mengelompokkan dB SPL ke zona warna yang sama dengan UI.
func ZoneOf(spl float64, listening bool) string {
	switch {
	case !listening:
		return "silent"
	case spl <= SafeSPL:
		return "safe"
	case spl <= LimitSPL:
		return "warn"
	}
	return "danger"
}

// EstimateSPL memperkirakan dB SPL yang sampai ke telinga.
func EstimateSPL(maxSPL float64, lvl Level, v Volume) float64 {
	gainDB := -100.0
	if v > 0 {
		gainDB = 20 * math.Log10(float64(v))
	}
	return maxSPL + float64(lvl) + gainDB
}
