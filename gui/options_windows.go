//go:build windows

package main

import (
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func getWindowsOptions() *windows.Options {
	return &windows.Options{
		WebviewUserDataFolder:     "SloPN",         // Store cache/data in SloPN subfolder
		WebviewBrowserCommandLine: "--disable-gpu", // Best compatibility for old Windows 10
	}
}
