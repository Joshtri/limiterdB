package domain

import "math"

const (
	MinTargetSPL = 40.0
	MaxTargetSPL = 100.0
	MinMaxSPL    = 60.0
	MaxMaxSPL    = 130.0
	MinCeiling   = 0.01
	// Boost speaker dalam persen amplitudo: 100% = tanpa boost, 200% = +6 dB.
	MinSpeakerBoost = 100.0
	MaxSpeakerBoost = 200.0
)

// BoostDB mengubah persen boost ke gain dB (200% ≈ +6 dB).
func BoostDB(pct float64) float64 {
	if pct <= MinSpeakerBoost {
		return 0
	}
	return 20 * math.Log10(pct/100)
}

// Settings adalah pengaturan user yang disimpan ke disk.
type Settings struct {
	LimiterEnabled bool    `json:"limiterEnabled"`
	TargetSPL      float64 `json:"targetSpl"` // batas dengar (dB SPL)
	Ceiling        float64 `json:"ceiling"`   // gain maksimum 0..1
	// SpeakerBoost: volume di atas 100% untuk speaker (lewat Equalizer APO).
	// Tidak pernah berlaku untuk headphone/headset.
	SpeakerBoost float64 `json:"speakerBoost"`
	// Calibrations: device ID → maxSPL hasil kalibrasi user.
	Calibrations map[string]float64 `json:"calibrations"`
}

func DefaultSettings() Settings {
	return Settings{
		LimiterEnabled: true,
		TargetSPL:      70,
		Ceiling:        1,
		SpeakerBoost:   MinSpeakerBoost,
		Calibrations:   map[string]float64{},
	}
}

// Normalize memperbaiki nilai yang hilang/di luar rentang (misal file
// settings lama atau diedit manual).
func (s Settings) Normalize() Settings {
	d := DefaultSettings()
	if !isFinite(s.TargetSPL) || s.TargetSPL == 0 {
		s.TargetSPL = d.TargetSPL
	}
	s.TargetSPL = clamp(s.TargetSPL, MinTargetSPL, MaxTargetSPL)
	if !isFinite(s.Ceiling) || s.Ceiling == 0 {
		s.Ceiling = d.Ceiling
	}
	s.Ceiling = clamp(s.Ceiling, MinCeiling, 1)
	if !isFinite(s.SpeakerBoost) {
		s.SpeakerBoost = d.SpeakerBoost
	}
	s.SpeakerBoost = clamp(s.SpeakerBoost, MinSpeakerBoost, MaxSpeakerBoost)

	cal := make(map[string]float64, len(s.Calibrations))
	for id, v := range s.Calibrations {
		if id != "" && isFinite(v) {
			cal[id] = clamp(v, MinMaxSPL, MaxMaxSPL)
		}
	}
	s.Calibrations = cal
	return s
}

// MaxSPLFor mengembalikan maxSPL untuk device: hasil kalibrasi kalau ada,
// selain itu default per jenis device.
func (s Settings) MaxSPLFor(d Device) (maxSPL float64, calibrated bool) {
	if v, ok := s.Calibrations[d.ID]; ok {
		return v, true
	}
	return DefaultMaxSPL(d.Kind), false
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

func isFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
