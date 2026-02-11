package main

import (
	"context"
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var icon []byte

var wailsApp *App

func main() {
	// Create an instance of the app structure
	wailsApp = NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "SloPN",
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
			initTray(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			wailsApp.Disconnect()
			wailsApp.shutdown(ctx)
		},
		Bind: []interface{}{
			wailsApp,
		},
		Windows: &windows.Options{
			WebviewUserDataFolder:    "SloPN", // Store webview data in SloPN folder
			WebviewBrowserCommandLine: "--disable-gpu", // Compatibility for old hardware
		},
		Menu: getAppMenu(),
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "SloPN",
				Message: "© 2026 webdunesurfer",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func getAppMenu() *menu.Menu {
	AppMenu := menu.NewMenu()
	if runtime.GOOS == "darwin" {
		AppMenu.Append(menu.AppMenu())
		AppMenu.Append(menu.EditMenu())
		AppMenu.Append(menu.WindowMenu())
	}
	return AppMenu
}
