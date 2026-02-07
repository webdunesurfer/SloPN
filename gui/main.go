package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

var wailsApp *App

func main() {
	// Create an instance of the app structure
	wailsApp = NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "SloPN VPN",
		Width:             800,
		Height:            650,
		DisableResize:     true,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 1},
		OnStartup: func(ctx context.Context) {
			wailsApp.startup(ctx)
			initTray("SloPN")
		},
		OnShutdown: func(ctx context.Context) {
			wailsApp.Disconnect()
			wailsApp.shutdown(ctx)
		},
		Bind: []interface{}{
			wailsApp,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}