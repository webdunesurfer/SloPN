//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	LogPath    = "C:\\ProgramData\\SloPN\\slopn-helper.log"
	SecretPath = "C:\\ProgramData\\SloPN\\ipc.secret"
)

func (h *Helper) setupDNS() {
	logHelper("[DNS] Configuring DNS for slopn-tap0...")
	// Force DNS to the VPN server's internal DNS
	cmd := exec.Command("netsh", "interface", "ip", "set", "dns", "name=slopn-tap0", "static", "10.100.0.1", "validate=no")
	if output, err := cmd.CombinedOutput(); err != nil {
		logHelper(fmt.Sprintf("[DNS] Error: %v (output: %s)", err, string(output)))
	} else {
		logHelper("[DNS] Success: DNS set to 10.100.0.1")
	}
}

func (h *Helper) restoreDNS() {
	logHelper("[DNS] Restoring DNS for slopn-tap0...")
	// Reset to DHCP or clear
	cmd := exec.Command("netsh", "interface", "ip", "set", "dns", "name=slopn-tap0", "source=dhcp")
	exec.Command("ipconfig", "/flushdns").Run()
	if output, err := cmd.CombinedOutput(); err != nil {
		logHelper(fmt.Sprintf("[DNS] Restore Error: %v (output: %s)", err, string(output)))
	}
}

func (h *Helper) getLogs() string {
	out, err := exec.Command("powershell", "-Command", fmt.Sprintf("Get-Content '%s' -Tail 100", LogPath)).Output()
	if err != nil {
		return "Failed to read logs: " + err.Error()
	}
	return string(out)
}

func (h *Helper) setupRouting(full bool, serverHost, serverVIP string) {
	if serverVIP == "" {
		return // Wait until VIP is known
	}

	if !full {
		logHelper(fmt.Sprintf("[VPN] Adding split-tunnel route for 10.100.0.0/24 via %s", serverVIP))
		exec.Command("route", "add", "10.100.0.0", "mask", "255.255.255.0", serverVIP).Run()
		return
	}
	
	logHelper("[VPN] Configuring Full Tunnel...")

	// 1. Add host route to the VPN server via the original gateway to prevent loops
	// We find the current gateway for the public internet
	gwOut, _ := exec.Command("powershell", "-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1).NextHop").Output()
	currentGW := strings.TrimSpace(string(gwOut))

	if currentGW != "" && currentGW != "0.0.0.0" {
		logHelper(fmt.Sprintf("[VPN] Original Gateway: %s. Pinning server route.", currentGW))
		exec.Command("route", "add", serverHost, currentGW).Run()
	}

	// 2. Override default gateway using the 0.0.0.0/1 and 128.0.0.0/1 trick
	// This is more reliable than deleting the default route
	logHelper("[VPN] Redirecting all traffic through TUN...")
	exec.Command("route", "add", "0.0.0.0", "mask", "128.0.0.0", serverVIP).Run()
	exec.Command("route", "add", "128.0.0.0", "mask", "128.0.0.0", serverVIP).Run()
	
	h.setupDNS()
}

func (h *Helper) cleanupRouting(full bool, serverHost string) {
	logHelper("[VPN] Cleaning up Windows routes...")
	
	if full {
		exec.Command("route", "delete", "0.0.0.0", "mask", "128.0.0.0").Run()
		exec.Command("route", "delete", "128.0.0.0", "mask", "128.0.0.0").Run()
		if serverHost != "" {
			exec.Command("route", "delete", serverHost).Run()
		}
		h.restoreDNS()
	}
	
	exec.Command("route", "delete", "10.100.0.0", "mask", "255.255.255.0").Run()
}