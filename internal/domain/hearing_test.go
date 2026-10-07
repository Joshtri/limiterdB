package domain

import (
	"math"
	"testing"
	"time"
)

func TestThresholdFor(t *testing.T) {
	tests := []struct {
		target, maxSPL, want float64
	}{
		{70, 100, -30},
		{80, 85, -5},
		{90, 85, 0},    // perangkat tidak bisa sekeras itu → 0 dBFS
		{40, 120, -60}, // di luar rentang limiter → -60
	}
	for _, tt := range tests {
		if got := float64(ThresholdFor(tt.target, tt.maxSPL)); got != tt.want {
			t.Errorf("ThresholdFor(%v, %v) = %v, want %v", tt.target, tt.maxSPL, got, tt.want)
		}
	}
}

func TestEstimateSPL(t *testing.T) {
	// headphone 100 dB, lagu -6 dBFS, volume -24 dB → ~70 dB
	got := EstimateSPL(100, -6, VolumeFromDB(-24))
	if math.Abs(got-70) > 0.01 {
		t.Fatalf("got %.2f, want 70", got)
	}
}

func TestSettingsNormalize(t *testing.T) {
	s := Settings{TargetSPL: 300, Ceiling: math.NaN(), Calibrations: map[string]float64{"a": 500, "": 90, "b": math.Inf(1)}}.Normalize()
	if s.TargetSPL != MaxTargetSPL || s.Ceiling != 1 {
		t.Fatalf("got target %v ceiling %v", s.TargetSPL, s.Ceiling)
	}
	if len(s.Calibrations) != 1 || s.Calibrations["a"] != MaxMaxSPL {
		t.Fatalf("calibrations not cleaned: %v", s.Calibrations)
	}

	if (Settings{}).Normalize().TargetSPL != DefaultSettings().TargetSPL {
		t.Fatal("empty settings should fall back to default target")
	}
}

func TestMaxSPLFor(t *testing.T) {
	s := DefaultSettings()
	s.Calibrations["dev1"] = 104
	if v, cal := s.MaxSPLFor(Device{ID: "dev1", Kind: "headphones"}); v != 104 || !cal {
		t.Fatalf("calibrated: got %v %v", v, cal)
	}
	if v, cal := s.MaxSPLFor(Device{ID: "dev2", Kind: "speakers"}); v != 85 || cal {
		t.Fatalf("default: got %v %v", v, cal)
	}
}

func TestExposureWeeklyDose(t *testing.T) {
	now := time.Date(2026, 10, 6, 20, 0, 0, 0, time.Local)
	l := NewExposureLog()

	// 80 dB selama 40 jam = 100% jatah mingguan
	l.Add(now, 40*time.Hour, 80)
	if got := l.Summary(now).Week.DosePct; math.Abs(got-100) > 0.01 {
		t.Fatalf("80 dB × 40 jam: got %.2f%%, want 100%%", got)
	}

	// +3 dB → energi 2×: 83 dB × 20 jam juga ≈ 100%
	l2 := NewExposureLog()
	l2.Add(now, 20*time.Hour, 83)
	if got := l2.Summary(now).Week.DosePct; math.Abs(got-99.5) > 0.5 {
		t.Fatalf("83 dB × 20 jam: got %.2f%%, want ≈100%%", got)
	}
}

func TestExposureSummaryWindowAndLeq(t *testing.T) {
	now := time.Date(2026, 10, 6, 20, 0, 0, 0, time.Local)
	l := NewExposureLog()
	l.Add(now, time.Hour, 70)
	l.Add(now, time.Hour, 70)
	l.Add(now.AddDate(0, 0, -3), 30*time.Minute, 80)
	l.Add(now.AddDate(0, 0, -7), 10*time.Hour, 90) // di luar 7 hari

	s := l.Summary(now)
	if len(s.Days) != 7 || s.Days[6].Date != "2026-10-06" || s.Days[0].Date != "2026-09-30" {
		t.Fatalf("window salah: %+v", s.Days)
	}
	if math.Abs(s.Today.LeqDB-70) > 0.01 || s.Today.ListenMin != 120 {
		t.Fatalf("today: %+v", s.Today)
	}
	if s.Week.ListenMin != 150 {
		t.Fatalf("week listen = %v, want 150", s.Week.ListenMin)
	}
	// Leq mingguan didominasi bagian 80 dB
	if s.Week.LeqDB <= 70 || s.Week.LeqDB >= 80 {
		t.Fatalf("week Leq = %.2f", s.Week.LeqDB)
	}

	l.Prune(now, 30)
	if len(l.Dates()) != 3 {
		t.Fatalf("prune 30 hari harus menyimpan semua: %v", l.Dates())
	}
	l.Prune(now, 5)
	if len(l.Dates()) != 2 {
		t.Fatalf("prune 5 hari: %v", l.Dates())
	}
}
