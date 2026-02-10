//go:build windows

package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func initTray(ctx context.Context) {
	appIcon := icon // Uses the embedded icon from main.go

	systemTrayMenu := menu.NewMenu()
	systemTrayMenu.AddText("Show SloPN", nil, func(_ *menu.CallbackData) {
		runtime.WindowShow(ctx)
		runtime.WindowUnminimise(ctx)
	})
	systemTrayMenu.AddSeparator()
	systemTrayMenu.AddText("About", nil, func(_ *menu.CallbackData) {
		wailsApp.ShowAbout()
	})
	systemTrayMenu.AddSeparator()
	systemTrayMenu.AddText("Quit", nil, func(_ *menu.CallbackData) {
		wailsApp.Disconnect()
		runtime.Quit(ctx)
	})

	runtime.SystemTraySetMenu(ctx, systemTrayMenu)
	runtime.SystemTraySetIcon(ctx, appIcon)
}

func updateTrayStatus(connected bool) {
	// Optional: Change icon or label based on connection
}
