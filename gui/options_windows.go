//go:build windows

package main

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func getWindowsOptions() *windows.Options {
	// Use %LOCALAPPDATA%/SloPN for webview data to ensure write permissions
	// and avoid issues when installed in Program Files.
	localAppData := os.Getenv("LOCALAPPDATA")
	userDataPath := filepath.Join(localAppData, "SloPN", "webview")

	return &windows.Options{
		WebviewUserDataPath:  userDataPath,
		WebviewGpuIsDisabled: true, // Best compatibility for old Windows 10
	}
}
