//go:build linux

package main

import (
	"context"
	"fmt"
)

func initTray(ctx context.Context) {
	fmt.Printf("[GUI] Tray initialized (Linux Tray todo)\n")
}

func updateTrayStatus(connected bool) {
}