//go:build linux

package main

import (
	"os"
	"path/filepath"
)

const (
	SecretPath = "/etc/slopn/ipc.secret"
	InstallConfigPath = "/etc/slopn/config.json"
	NewInstallMarkerPath = "/etc/slopn/.new_install"
)

func getConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "slopn")
}