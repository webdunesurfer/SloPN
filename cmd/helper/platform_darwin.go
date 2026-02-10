//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	LogPath    = "/var/log/slopn-helper.log"
	SecretPath = "/Library/Application Support/SloPN/ipc.secret"
)

func (h *Helper) getAllActiveInterfaces() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return []string{"Wi-Fi"}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var active []string
	for _, line := range lines {
		if strings.Contains(line, "*") {
			continue // Skip disabled services
		}
		if line == "An asterisk (*) denotes that a network service is disabled." {
			continue
		}
		active = append(active, line)
	}
	return active
}

func (h *Helper) setupDNS() {
	interfaces := h.getAllActiveInterfaces()
	logHelper(fmt.Sprintf("[DNS] Protecting %d interfaces...", len(interfaces)))

	for _, iface := range interfaces {
		logHelper(fmt.Sprintf("[DNS] Forcing SloPN Internal DNS on %s...", iface))
		exec.Command("networksetup", "-setdnsservers", iface, "10.100.0.1").Run()
	}
	
	exec.Command("dscacheutil", "-flushcache").Run()
	exec.Command("killall", "-HUP", "mDNSResponder").Run()
}

func (h *Helper) restoreDNS() {
	interfaces := h.getAllActiveInterfaces()
	logHelper("[DNS] Restoring settings for all interfaces...")
	for _, iface := range interfaces {
		exec.Command("networksetup", "-setdnsservers", iface, "Empty").Run()
	}
	exec.Command("dscacheutil", "-flushcache").Run()
}

func (h *Helper) getLogs() string {
	out, err := exec.Command("tail", "-n", "100", LogPath).Output()
	if err != nil {
		return "Failed to read logs: " + err.Error()
	}
	return string(out)
}

func (h *Helper) setupRouting(full bool, serverHost, serverVIP, ifceName string) {
	// 1. Always ensure we have a host route to the VPN server via the physical gateway
	if serverHost != "" {
		gwOut, _ := exec.Command("sh", "-c", "route -n get default | awk '/gateway: / {print $2}'").Output()
		currentGW := strings.TrimSpace(string(gwOut))
		if currentGW != "" {
			logHelper(fmt.Sprintf("[VPN] Ensuring host route for %s via %s", serverHost, currentGW))
			exec.Command("route", "add", "-host", serverHost, currentGW).Run()
		}
	}

	if !full || serverVIP == "" {
		return
	}

	logHelper("[VPN] Configuring Full Tunnel (v4 + v6 protection)...")
	h.setupDNS()

	// Use the "more specific route" trick (0.0.0.0/1 and 128.0.0.0/1)
	logHelper(fmt.Sprintf("[VPN] Redirecting traffic via %s...", serverVIP))
	exec.Command("route", "add", "-net", "0.0.0.0/1", serverVIP).Run()
	exec.Command("route", "add", "-net", "128.0.0.0/1", serverVIP).Run()

	logHelper("[VPN] Routing table updated.")
}

func (h *Helper) cleanupRouting(full bool, serverHost, ifceName string) {
	logHelper("[VPN] Cleaning up routing...")

	if full {
		exec.Command("route", "delete", "-net", "0.0.0.0/1").Run()
		exec.Command("route", "delete", "-net", "128.0.0.0/1").Run()
		h.restoreDNS()
	}

	if serverHost != "" {
		exec.Command("route", "delete", "-host", serverHost).Run()
		logHelper(fmt.Sprintf("[VPN] Removed host route for: %s", serverHost))
	}
}
