package domain

import (
	"math"
	"testing"
)

func TestNextVolume(t *testing.T) {
	l, _ := NewLimiter(1.0, 0.02, 0.05)
	th, _ := NewThreshold(-20)

	tests := []struct {
		name    string
		cur     Volume
		lvl     Level
		ceiling Volume
		want    float64
	}{
		{"lagu keras: turun ke volume ideal", 1.0, -6, 1.0, 0.1995},
		{"lagu pelan: naik sebatas release step", 0.5, -40, 1.0, 0.52},
		{"naik tidak melewati target", 0.5, -14, 1.0, 0.5012}, // target VolumeFromDB(-6dB)=0.5012 < cur+releaseStep(0.52) → clamp ke target
		{"ceiling dihormati", 0.5, -80, 0.6, 0.52},
		{"lantai minVolume", 0.5, 0, 1.0, 0.1}, // target 0.1 > min 0.05
		{"sudah pas: tidak berubah", 0.1995, -6, 1.0, 0.1995},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float64(l.NextVolume(tt.cur, tt.lvl, th, tt.ceiling))
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("got %.4f, want %.4f", got, tt.want)
			}
		})
	}
}

func TestNewThresholdRejectsInvalid(t *testing.T) {
	for _, db := range []float64{1, -61, math.NaN()} {
		if _, err := NewThreshold(db); err == nil {
			t.Errorf("expected error for %v", db)
		}
	}
}
