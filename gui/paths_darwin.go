//go:build darwin

package main

import (
	"os"
	"path/filepath"
)

const (
	SecretPath = "/Library/Application Support/SloPN/ipc.secret"
	InstallConfigPath = "/Library/Application Support/SloPN/config.json"
	NewInstallMarkerPath = "/Library/Application Support/SloPN/.new_install"
)

func getConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "SloPN")
}
