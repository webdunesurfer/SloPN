//go:build windows

package main

import "fmt"

func initTray(title string) {
	fmt.Printf("[GUI] Tray initialized with title: %s (Windows Tray todo)
", title)
}

func updateTrayStatus(connected bool) {
	// Wails usually handles window management. 
	// We can implement actual Windows NotifyIcon logic here later if needed.
}
