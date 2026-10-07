// Package tray menampilkan ikon limiterdB di system tray Windows.
package tray

import (
	"runtime"
	"sync"

	"github.com/getlantern/systray"
)

// Handlers dipanggil dari goroutine tray saat menu diklik.
type Handlers struct {
	OnOpen   func()
	OnToggle func()
	OnQuit   func()
}

// State adalah tampilan tray: warna ikon, status limiter, dan tooltip.
type State struct {
	Zone    string // silent | safe | warn | danger
	Enabled bool
	Tooltip string
}

type Tray struct {
	h Handlers

	mu      sync.Mutex
	ready   bool
	last    State
	pending *State
	toggle  *systray.MenuItem
}

// Start menjalankan tray di OS thread sendiri. Message loop Windows bersifat
// per-thread, jadi tray tidak mengganggu loop milik Wails.
func Start(h Handlers) *Tray {
	t := &Tray{h: h}
	go func() {
		runtime.LockOSThread()
		systray.Run(t.onReady, nil)
	}()
	return t
}

func (t *Tray) onReady() {
	mOpen := systray.AddMenuItem("Buka limiterdB", "Tampilkan jendela")
	t.toggle = systray.AddMenuItem("Limiter aktif", "Nyalakan/matikan penyesuaian volume otomatis")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Keluar", "Tutup limiterdB sepenuhnya")

	t.mu.Lock()
	t.ready = true
	initial := State{Zone: "silent", Tooltip: "limiterdB"}
	if t.pending != nil {
		initial = *t.pending
	}
	t.mu.Unlock()
	t.apply(initial, true)

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				call(t.h.OnOpen)
			case <-t.toggle.ClickedCh:
				call(t.h.OnToggle)
			case <-mQuit.ClickedCh:
				call(t.h.OnQuit)
				return
			}
		}
	}()
}

func call(fn func()) {
	if fn != nil {
		fn()
	}
}

// Update mengubah ikon/tooltip/centang menu. Aman dipanggil dari goroutine
// mana pun; tidak melakukan apa-apa kalau tidak ada perubahan.
func (t *Tray) Update(s State) {
	t.mu.Lock()
	if !t.ready {
		t.pending = &s
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()
	t.apply(s, false)
}

func (t *Tray) apply(s State, force bool) {
	t.mu.Lock()
	prev := t.last
	t.last = s
	t.mu.Unlock()

	if force || prev.Zone != s.Zone || prev.Enabled != s.Enabled {
		systray.SetIcon(makeIcon(s.Zone, s.Enabled))
	}
	if force || prev.Tooltip != s.Tooltip {
		systray.SetTooltip(s.Tooltip)
	}
	if force || prev.Enabled != s.Enabled {
		if s.Enabled {
			t.toggle.Check()
		} else {
			t.toggle.Uncheck()
		}
	}
}

// Stop menghapus ikon tray.
func (t *Tray) Stop() { systray.Quit() }
