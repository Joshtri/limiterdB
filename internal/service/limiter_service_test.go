package service

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"limiterdB/internal/domain"
)

type fakeMeter struct {
	mu  sync.Mutex
	lvl domain.Level
}

func (m *fakeMeter) Read(context.Context) (domain.Level, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lvl, nil
}

type fakeVolume struct {
	mu   sync.Mutex
	v    domain.Volume
	sets int
}

func (f *fakeVolume) Get() (domain.Volume, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.v, nil
}

func (f *fakeVolume) Set(v domain.Volume) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.v = v
	f.sets++
	return nil
}

type fakeDevices struct{ d domain.Device }

func (f *fakeDevices) Device() (domain.Device, error) { return f.d, nil }

type fakeStore struct {
	mu    sync.Mutex
	saved []domain.ExposureLog
}

func (f *fakeStore) LoadExposure() (domain.ExposureLog, error) { return domain.NewExposureLog(), nil }
func (f *fakeStore) SaveExposure(l domain.ExposureLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, l)
	return nil
}

var headphones = domain.Device{ID: "hp", Name: "Test HP", Kind: "headphones"} // default maxSPL 100

func newTestService(t *testing.T, m *fakeMeter, v *fakeVolume, st domain.Settings) (*LimiterService, *fakeStore) {
	t.Helper()
	l, err := domain.NewLimiter(1.0, 0.02, domain.NewVolume(0.01))
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{}
	s := NewLimiterService(l, m, v, &fakeDevices{d: headphones}, store, st, domain.NewExposureLog(), 5*time.Millisecond)
	s.refreshDevice()
	return s, store
}

func settings(enabled bool, target float64) domain.Settings {
	st := domain.DefaultSettings()
	st.LimiterEnabled = enabled
	st.TargetSPL = target
	return st
}

func TestDisabledLimiterMonitorsWithoutTouchingVolume(t *testing.T) {
	m := &fakeMeter{lvl: -12}
	v := &fakeVolume{v: 0.7}
	s, _ := newTestService(t, m, v, settings(false, 70))

	s.tick(context.Background(), time.Now(), 50*time.Millisecond)

	st := s.Status()
	if st.Enabled || st.LastLevel != -12 || st.CurrentVolume != 0.7 {
		t.Fatalf("status: %+v", st)
	}
	if v.sets != 0 {
		t.Fatalf("limiter mati tidak boleh mengubah volume, got %d Set", v.sets)
	}
	if !st.Listening || st.MaxSPL != 100 || st.Threshold != -30 {
		t.Fatalf("status: %+v", st)
	}
}

func TestEnabledLimiterBringsLoudSongToTarget(t *testing.T) {
	m := &fakeMeter{lvl: -6} // lagu keras
	v := &fakeVolume{v: 1.0}
	s, _ := newTestService(t, m, v, settings(true, 70)) // threshold 70-100 = -30

	s.tick(context.Background(), time.Now(), 50*time.Millisecond)

	got, _ := v.Get()
	// -30 - (-6) = -24 dB → gain ≈ 0.0631
	if math.Abs(float64(got)-0.0631) > 0.001 {
		t.Fatalf("volume %.4f, want ≈0.0631", got)
	}
	if heard := s.Status().HeardSPL; math.Abs(heard-70) > 0.1 {
		t.Fatalf("heard %.2f, want ≈70", heard)
	}
}

func TestExposureCountsOnlyAudibleTicks(t *testing.T) {
	m := &fakeMeter{lvl: -20}
	v := &fakeVolume{v: 1.0}
	s, _ := newTestService(t, m, v, settings(false, 70))
	now := time.Now()

	for i := 0; i < 20; i++ { // 1 detik di 80 dB (100 - 20 + 0)
		s.tick(context.Background(), now, 50*time.Millisecond)
	}
	m.lvl = -100 // senyap
	for i := 0; i < 20; i++ {
		s.tick(context.Background(), now, 50*time.Millisecond)
	}

	today := s.Exposure().Today
	if math.Abs(today.ListenMin*60-1) > 1e-6 {
		t.Fatalf("listen = %.3f s, want 1 s", today.ListenMin*60)
	}
	if math.Abs(today.LeqDB-80) > 0.01 {
		t.Fatalf("Leq = %.2f, want 80", today.LeqDB)
	}
}

func TestSettersNormalizeAndCalibrateCurrentDevice(t *testing.T) {
	s, _ := newTestService(t, &fakeMeter{lvl: -100}, &fakeVolume{v: 1}, settings(true, 70))

	if got := s.SetTargetSPL(500).TargetSPL; got != domain.MaxTargetSPL {
		t.Fatalf("target not clamped: %v", got)
	}
	st, err := s.SetCalibration(104)
	if err != nil || st.Calibrations["hp"] != 104 {
		t.Fatalf("calibration: %v %v", st.Calibrations, err)
	}
	if s.Status().MaxSPL != 104 || !s.Status().Calibrated {
		t.Fatal("status should use calibration")
	}
	if _, ok := s.ResetCalibration().Calibrations["hp"]; ok {
		t.Fatal("reset should remove calibration")
	}

	// Settings() harus salinan, bukan map internal
	c := s.Settings()
	c.Calibrations["x"] = 1
	if _, ok := s.Settings().Calibrations["x"]; ok {
		t.Fatal("Settings() leaked internal map")
	}
}

func TestStopSavesExposure(t *testing.T) {
	s, store := newTestService(t, &fakeMeter{lvl: -20}, &fakeVolume{v: 1}, settings(false, 70))
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	s.Stop()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.saved) == 0 || len(store.saved[len(store.saved)-1].Days) == 0 {
		t.Fatal("Stop harus menyimpan catatan paparan")
	}
}

func TestLimiterPausedOnSpeakers(t *testing.T) {
	m := &fakeMeter{lvl: -6} // lagu keras
	v := &fakeVolume{v: 1.0}
	l, _ := domain.NewLimiter(1.0, 0.02, domain.NewVolume(0.01))
	speakers := domain.Device{ID: "spk", Name: "Test Speaker", Kind: "speakers"}
	s := NewLimiterService(l, m, v, &fakeDevices{d: speakers}, nil, settings(true, 70), domain.NewExposureLog(), 5*time.Millisecond)
	s.refreshDevice()

	s.tick(context.Background(), time.Now(), 50*time.Millisecond)

	if v.sets != 0 {
		t.Fatalf("limiter tidak boleh mengubah volume speaker, got %d Set", v.sets)
	}
	if st := s.Status(); !st.Enabled || st.Active {
		t.Fatalf("speaker: want enabled && !active, got %+v", st)
	}
}

type fakeBooster struct {
	mu    sync.Mutex
	calls []float64
}

func (f *fakeBooster) SetBoost(db float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, db)
	return nil
}

func TestSpeakerBoostOnlyOnSpeakers(t *testing.T) {
	devs := &fakeDevices{d: domain.Device{ID: "spk", Kind: "speakers"}}
	l, _ := domain.NewLimiter(1.0, 0.02, domain.NewVolume(0.01))
	st := settings(true, 70)
	st.SpeakerBoost = 200
	s := NewLimiterService(l, &fakeMeter{lvl: -20}, &fakeVolume{v: 1}, devs, nil, st, domain.NewExposureLog(), 5*time.Millisecond)
	b := &fakeBooster{}
	s.UseBooster(b)

	s.refreshDevice()
	s.syncBoost()
	s.syncBoost() // tidak berubah → tidak ditulis ulang
	if len(b.calls) != 1 || math.Abs(b.calls[0]-6.02) > 0.01 {
		t.Fatalf("speaker 200%%: calls %v, want [≈6.02]", b.calls)
	}
	if got := s.Status().BoostDB; math.Abs(got-6.02) > 0.01 {
		t.Fatalf("status boost %v", got)
	}

	devs.d = headphones // colok headphone → boost harus 0
	s.refreshDevice()
	s.syncBoost()
	if last := b.calls[len(b.calls)-1]; last != 0 {
		t.Fatalf("headphone: boost %v, want 0", last)
	}
}

func TestBoostDB(t *testing.T) {
	if domain.BoostDB(100) != 0 || math.Abs(domain.BoostDB(200)-6.02) > 0.01 {
		t.Fatalf("BoostDB salah: %v %v", domain.BoostDB(100), domain.BoostDB(200))
	}
	if (domain.Settings{}).Normalize().SpeakerBoost != 100 {
		t.Fatal("settings lama (tanpa speakerBoost) harus jadi 100%")
	}
}
