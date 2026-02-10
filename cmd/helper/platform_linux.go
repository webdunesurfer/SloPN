//go:build linux

package main

import (
	"fmt"
	"os/exec"
)

const (
	LogPath    = "/var/log/slopn-helper.log"
	SecretPath = "/etc/slopn/ipc.secret"
)

func (h *Helper) setupDNS() {
	// Typically managed via resolvconf or systemd-resolved on Linux
}

func (h *Helper) restoreDNS() {
}

func (h *Helper) getLogs() string {
	out, err := exec.Command("tail", "-n", "100", LogPath).Output()
	if err != nil {
		return "Failed to read logs: " + err.Error()
	}
	return string(out)
}

func (h *Helper) setupRouting(full bool, serverHost, serverVIP, ifceName string) {
}

func (h *Helper) cleanupRouting(full bool, serverHost, ifceName string) {
}