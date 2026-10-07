package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"limiterdB/internal/adapter"
	"limiterdB/internal/domain"
	"limiterdB/internal/service"
	"limiterdB/internal/store"
	"limiterdB/internal/tray"
)

const (
	tickInterval = 50 * time.Millisecond
	// trayTick: seberapa sering status tray dihitung; ikon/tooltip di-update
	// tiap trayUpdateEvery tick.
	trayTick        = 250 * time.Millisecond
	trayUpdateEvery = 4
	// trayAvgTau: rata-rata ±2 detik supaya warna ikon tidak berkedip.
	trayAvgTau = 2 * time.Second
)

var errNotReady = errors.New("limiter belum siap")

// App struct
type App struct {
	ctx     context.Context
	wasapi  *adapter.Adapter
	svc     *service.LimiterService
	store   *store.Store
	tray    *tray.Tray
	initErr error
}

// SettingsView adalah pengaturan yang ditampilkan di UI.
type SettingsView struct {
	LimiterEnabled bool    `json:"limiterEnabled"`
	TargetSPL      float64 `json:"targetSpl"`
	Ceiling        float64 `json:"ceiling"`
	AutoStart      bool    `json:"autoStart"`
	DataDir        string  `json:"dataDir"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	settings := domain.DefaultSettings()
	exposure := domain.NewExposureLog()
	var exposureStore domain.ExposureStore // nil interface kalau penyimpanan gagal
	if dir, err := store.DefaultDir(); err == nil {
		if st, err := store.New(dir); err == nil {
			a.store = st
			exposureStore = st
			// File rusak → tetap jalan dengan default (Load mengembalikan default).
			settings, _ = st.LoadSettings()
			exposure, _ = st.LoadExposure()
		}
	}

	wasapiAdapter, err := adapter.NewAdapter()
	if err != nil {
		a.initErr = err
		return
	}
	a.wasapi = wasapiAdapter

	// minVolume 0.01 = −40 dB: cukup rendah untuk meredam lagu yang sangat
	// keras ke target dengar yang pelan, tapi tidak pernah mute total.
	limiter, err := domain.NewLimiter(0.5, 0.001, domain.NewVolume(0.01))
	if err != nil {
		a.initErr = err
		return
	}

	a.svc = service.NewLimiterService(limiter, wasapiAdapter, wasapiAdapter, wasapiAdapter,
		exposureStore, settings, exposure, tickInterval)
	if err := a.svc.Start(ctx); err != nil {
		a.initErr = err
		return
	}

	a.tray = tray.Start(tray.Handlers{
		OnOpen:   a.showWindow,
		OnToggle: func() { _ = a.SetLimiterEnabled(!a.svc.Settings().LimiterEnabled) },
		OnQuit:   func() { wruntime.Quit(a.ctx) },
	})
	go a.trayLoop(ctx)
}

// shutdown dipanggil saat aplikasi benar-benar keluar (menu tray "Keluar").
// Urutannya penting: hentikan loop limiter dulu (sekaligus menyimpan catatan
// paparan) sebelum menutup adapter WASAPI.
func (a *App) shutdown(ctx context.Context) {
	if a.svc != nil {
		a.svc.Stop()
		a.persist(a.svc.Settings())
	}
	if a.wasapi != nil {
		a.wasapi.Close()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
}

// showWindow menampilkan jendela dari tray, atau saat limiterdB dibuka lagi
// padahal sudah jalan (single instance).
func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowShow(a.ctx)
	wruntime.WindowUnminimise(a.ctx)
}

// trayLoop memperbarui warna ikon dan tooltip tray dari status limiter.
func (a *App) trayLoop(ctx context.Context) {
	ticker := time.NewTicker(trayTick)
	defer ticker.Stop()

	alpha := 1 - math.Exp(-float64(trayTick)/float64(trayAvgTau))
	energy := 0.0 // rata-rata 10^(dB/10)
	n := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		st := a.svc.Status()
		e := 0.0
		if st.Listening {
			e = math.Pow(10, st.HeardSPL/10)
		}
		energy = (1-alpha)*energy + alpha*e

		if n++; n%trayUpdateEvery != 0 {
			continue
		}

		listening := st.Listening && energy > 0
		avg := 0.0
		if energy > 0 {
			avg = 10 * math.Log10(energy)
		}
		zone := domain.ZoneOf(avg, listening)
		a.tray.Update(tray.State{Zone: zone, Enabled: st.Enabled, Tooltip: trayTooltip(st, zone, avg, a.svc.Exposure())})
	}
}

var zoneLabel = map[string]string{
	"silent": "Senyap",
	"safe":   "Aman",
	"warn":   "Waspada",
	"danger": "Terlalu keras",
}

func trayTooltip(st service.Status, zone string, avg float64, exp domain.ExposureSummary) string {
	now := zoneLabel[zone]
	if zone != "silent" {
		now = fmt.Sprintf("~%.0f dB · %s", avg, now)
	}
	limiter := "Limiter mati"
	if st.Enabled {
		limiter = fmt.Sprintf("Limiter aktif (%.0f dB)", st.TargetSPL)
	}
	// Tooltip tray Windows maksimal 127 karakter.
	return fmt.Sprintf("limiterdB — %s\n7 hari: %.0f%% batas aman\n%s", now, exp.Week.DosePct, limiter)
}

func (a *App) persist(st domain.Settings) {
	if a.store != nil {
		_ = a.store.SaveSettings(st)
	}
}

func (a *App) ready() error {
	if a.initErr != nil {
		return a.initErr
	}
	if a.svc == nil {
		return errNotReady
	}
	return nil
}

// --- method yang dipanggil dari UI ---

// Status mengembalikan snapshot kondisi limiter saat ini.
func (a *App) Status() (service.Status, error) {
	if err := a.ready(); err != nil {
		return service.Status{}, err
	}
	return a.svc.Status(), nil
}

// Exposure mengembalikan ringkasan paparan hari ini dan 7 hari terakhir.
func (a *App) Exposure() (domain.ExposureSummary, error) {
	if err := a.ready(); err != nil {
		return domain.ExposureSummary{}, err
	}
	return a.svc.Exposure(), nil
}

// Settings mengembalikan pengaturan yang tersimpan.
func (a *App) Settings() (SettingsView, error) {
	if err := a.ready(); err != nil {
		return SettingsView{}, err
	}
	st := a.svc.Settings()
	auto, _ := adapter.AutoStartEnabled()
	view := SettingsView{
		LimiterEnabled: st.LimiterEnabled,
		TargetSPL:      st.TargetSPL,
		Ceiling:        st.Ceiling,
		AutoStart:      auto,
	}
	if a.store != nil {
		view.DataDir = a.store.Dir()
	}
	return view, nil
}

// SetLimiterEnabled menyalakan/mematikan penyesuaian volume otomatis.
func (a *App) SetLimiterEnabled(on bool) error {
	if err := a.ready(); err != nil {
		return err
	}
	a.persist(a.svc.SetEnabled(on))
	return nil
}

// SetTargetSPL mengubah batas dengar (dB SPL).
func (a *App) SetTargetSPL(spl float64) error {
	if err := a.ready(); err != nil {
		return err
	}
	a.persist(a.svc.SetTargetSPL(spl))
	return nil
}

// SetCeiling mengubah gain maksimum (0..1).
func (a *App) SetCeiling(v float64) error {
	if err := a.ready(); err != nil {
		return err
	}
	a.persist(a.svc.SetCeiling(v))
	return nil
}

// SetCalibration menyimpan kerasnya device aktif di volume 100% (dB SPL).
func (a *App) SetCalibration(maxSPL float64) error {
	if err := a.ready(); err != nil {
		return err
	}
	st, err := a.svc.SetCalibration(maxSPL)
	if err != nil {
		return err
	}
	a.persist(st)
	return nil
}

// ResetCalibration kembali ke perkiraan default untuk device aktif.
func (a *App) ResetCalibration() error {
	if err := a.ready(); err != nil {
		return err
	}
	a.persist(a.svc.ResetCalibration())
	return nil
}

// SetAutoStart mendaftarkan limiterdB untuk jalan otomatis saat login.
func (a *App) SetAutoStart(on bool) error {
	return adapter.SetAutoStart(on)
}
