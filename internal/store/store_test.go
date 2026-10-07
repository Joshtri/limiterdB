package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"limiterdB/internal/domain"
)

func TestSettingsRoundTripAndDefaults(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.LoadSettings()
	if err != nil || got.TargetSPL != domain.DefaultSettings().TargetSPL {
		t.Fatalf("missing file should give defaults: %+v %v", got, err)
	}

	want := domain.DefaultSettings()
	want.LimiterEnabled = false
	want.TargetSPL = 64
	want.Ceiling = 0.5
	want.Calibrations["dev"] = 103.5
	if err := s.SaveSettings(want); err != nil {
		t.Fatal(err)
	}
	got, err = s.LoadSettings()
	if err != nil || got.LimiterEnabled || got.TargetSPL != 64 || got.Ceiling != 0.5 || got.Calibrations["dev"] != 103.5 {
		t.Fatalf("round trip: %+v %v", got, err)
	}
}

func TestCorruptSettingsFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadSettings()
	if err == nil {
		t.Fatal("expected error for corrupt file")
	}
	if got.TargetSPL != domain.DefaultSettings().TargetSPL {
		t.Fatalf("corrupt file should still return defaults: %+v", got)
	}
}

func TestExposureRoundTrip(t *testing.T) {
	s, _ := New(t.TempDir())
	l := domain.NewExposureLog()
	now := time.Now()
	l.Add(now, time.Hour, 75)
	if err := s.SaveExposure(l); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadExposure()
	if err != nil {
		t.Fatal(err)
	}
	if a, b := got.Summary(now).Today, l.Summary(now).Today; a != b {
		t.Fatalf("round trip: %+v vs %+v", a, b)
	}
}
