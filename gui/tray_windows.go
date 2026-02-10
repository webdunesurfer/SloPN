//go:build windows

package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func getSystemTray() *options.SystemTray {
	systemTrayMenu := menu.NewMenu()
	systemTrayMenu.AddText("Show SloPN", nil, func(_ *menu.CallbackData) {
		if wailsApp != nil && wailsApp.ctx != nil {
			runtime.WindowShow(wailsApp.ctx)
			runtime.WindowUnminimise(wailsApp.ctx)
		}
	})
	systemTrayMenu.AddSeparator()
	systemTrayMenu.AddText("About", nil, func(_ *menu.CallbackData) {
		if wailsApp != nil {
			wailsApp.ShowAbout()
		}
	})
	systemTrayMenu.AddSeparator()
	systemTrayMenu.AddText("Quit", nil, func(_ *menu.CallbackData) {
		if wailsApp != nil {
			wailsApp.Disconnect()
			if wailsApp.ctx != nil {
				runtime.Quit(wailsApp.ctx)
			}
		}
	})

	return &options.SystemTray{
		Title: "SloPN",
		Icon:  icon,
		Menu:  systemTrayMenu,
		OnLeftClick: func() {
			if wailsApp != nil && wailsApp.ctx != nil {
				runtime.WindowShow(wailsApp.ctx)
				runtime.WindowUnminimise(wailsApp.ctx)
			}
		},
	}
}

func initTray(ctx context.Context) {
	// Already handled by getSystemTray in wails.Run
}

func updateTrayStatus(connected bool) {
	// Optional: Change icon based on state
}
