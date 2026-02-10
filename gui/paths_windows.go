//go:build windows

package main

import (
	"os"
	"path/filepath"
)

const (
	SecretPath = `C:\ProgramData\SloPN\ipc.secret`
	InstallConfigPath = `C:\ProgramData\SloPN\config.json`
)

func getConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "SloPN")
}