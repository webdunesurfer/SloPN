//go:build linux

package main

import "fmt"

func initTray(title string) {
	fmt.Printf("[GUI] Tray initialized with title: %s (Linux Tray todo)
", title)
}

func updateTrayStatus(connected bool) {
}
