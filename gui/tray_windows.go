//go:build windows

package main

import (
	"context"
	"fmt"
)

func initTray(ctx context.Context) {
	fmt.Println("Windows Tray not yet implemented (requires Win32 API or external lib)")
}

func updateTrayStatus(connected bool) {
	// Placeholder
}