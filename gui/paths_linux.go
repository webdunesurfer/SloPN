//go:build linux

package main

import (
	"os"
	"path/filepath"
)

const (
	SecretPath = "/etc/slopn/ipc.secret"
	InstallConfigPath = "/etc/slopn/config.json"
)

func getConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "slopn")
}
