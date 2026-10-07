package main

import (
	"embed"
	"os"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"limiterdB/internal/adapter"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "limiterdB",
		Width:     720,
		Height:    840,
		MinWidth:  600,
		MinHeight: 560,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Sama dengan --color-bg di frontend supaya tidak ada kedipan warna saat dibuka.
		BackgroundColour: &options.RGBA{R: 25, G: 32, B: 41, A: 255},
		Windows: &windows.Options{
			// Zoom bawaan WebView2 (Ctrl+scroll, pinch touchpad) dimatikan; zoom hanya
			// lewat Ctrl+ / Ctrl- / Ctrl+0 yang ditangani frontend (lib/desktop.js).
			IsZoomControlEnabled: false,
			DisablePinchZoom:     true,
			// Title bar gelap menyatu dengan header aplikasi.
			Theme: windows.Dark,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:          windows.RGB(25, 32, 41),
				DarkModeTitleBarInactive:  windows.RGB(25, 32, 41),
				DarkModeTitleText:         windows.RGB(236, 239, 242),
				DarkModeTitleTextInactive: windows.RGB(125, 128, 134),
				DarkModeBorder:            windows.RGB(59, 67, 77),
				DarkModeBorderInactive:    windows.RGB(44, 52, 61),
			},
		},
		// Dijalankan otomatis saat login → langsung ke tray tanpa jendela.
		StartHidden: slices.Contains(os.Args[1:], adapter.MinimizedFlag),
		// Tombol X hanya menyembunyikan jendela; limiter tetap jalan di tray.
		// Keluar sepenuhnya lewat menu tray "Keluar".
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "6f0c3b9e-2a41-4d8e-9a57-limiterdB",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) { app.showWindow() },
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
