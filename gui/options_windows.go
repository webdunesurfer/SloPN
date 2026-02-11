//go:build windows

package main

import (
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func getWindowsOptions() *windows.Options {
	return &windows.Options{
		WebviewUserDataPath:  "SloPN", // Correct field name for data folder
		WebviewGpuIsDisabled: true,    // Correct field name to disable GPU
	}
}