//go:build windows

package adapter

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
)

// Smoke test terhadap audio device asli. Jalankan dengan:
//
//	LIMITERDB_HW=1 go test ./internal/adapter -run Hardware -v
func TestHardwareSmoke(t *testing.T) {
	if os.Getenv("LIMITERDB_HW") == "" {
		t.Skip("set LIMITERDB_HW=1 untuk menjalankan test hardware")
	}

	a, err := NewAdapter()
	if err != nil {
		t.Fatal(err)
	}

	info, err := a.Device()
	if err != nil {
		t.Fatalf("Device: %v", err)
	}
	t.Logf("device: %+v", info)
	if info.ID == "" || info.MinDB >= info.MaxDB {
		t.Fatalf("device info tidak valid: %+v", info)
	}

	lvl, err := a.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	t.Logf("level: %.1f dBFS", lvl)

	// Tulis ulang volume yang sama → harus kembali sama (dalam 0.5 dB).
	v, err := a.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := a.Set(v); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v2, _ := a.Get()
	t.Logf("gain: %.4f (%.2f dB) → %.4f", v, 20*math.Log10(float64(v)), v2)
	if v > 0 && math.Abs(20*math.Log10(float64(v2)/float64(v))) > 0.5 {
		t.Fatalf("round-trip volume meleset: %.4f → %.4f", v, v2)
	}

	a.Close()
	if _, err := a.Device(); !errors.Is(err, ErrClosed) {
		t.Fatalf("setelah Close: got %v, want ErrClosed", err)
	}
	a.Close() // Close kedua tidak boleh panic
}
