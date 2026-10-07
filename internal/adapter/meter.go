//go:build windows

package adapter

import (
	"context"

	"limiterdB/internal/domain"
)

// Read mengambil peak amplitude terkini dari default render endpoint dan
// mengonversinya ke domain.Level (dBFS). Mengimplementasikan
// domain.LevelMeter.
func (a *Adapter) Read(ctx context.Context) (domain.Level, error) {
	var peak float32
	if err := a.exec(func(e *endpoint) error {
		return e.meter.GetPeakValue(&peak)
	}); err != nil {
		return 0, err
	}
	return domain.LevelFromAmplitude(float64(peak)), nil
}
