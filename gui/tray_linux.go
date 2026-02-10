//go:build linux

package main

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/options"
)

func getSystemTray() *options.SystemTray {
	return nil
}

func initTray(ctx context.Context) {
	fmt.Printf("[GUI] Tray initialized (Linux Tray todo)\n")
}

func updateTrayStatus(connected bool) {
}
