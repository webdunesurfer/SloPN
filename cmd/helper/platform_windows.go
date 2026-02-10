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
	cmd := exec.Command("netsh", "interface", "ip", "set", "dns", "name=slopn-tap0", "static", "10.100.0.1", "validate=no")
	if output, err := cmd.CombinedOutput(); err != nil {
		logHelper(fmt.Sprintf("[DNS] Error: %v (output: %s)", err, string(output)))
	} else {
		logHelper("[DNS] Success: DNS set to 10.100.0.1")
	}
}

func (h *Helper) restoreDNS() {
	logHelper("[DNS] Restoring DNS for slopn-tap0...")
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

func (h *Helper) getInterfaceIndex(name string) string {
	out, err := exec.Command("powershell", "-Command", fmt.Sprintf("(Get-NetAdapter -Name '%s').ifIndex", name)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (h *Helper) setupRouting(full bool, serverHost, serverVIP string) {
	if serverVIP == "" {
		return
	}

	ifIndex := h.getInterfaceIndex("slopn-tap0")
	if ifIndex == "" {
		logHelper("[VPN] Error: Could not find interface index for slopn-tap0")
		return
	}

	if !full {
		logHelper(fmt.Sprintf("[VPN] Adding split-tunnel route for 10.100.0.0/24 via %s (IF %s)", serverVIP, ifIndex))
		exec.Command("route", "add", "10.100.0.0", "mask", "255.255.255.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
		return
	}
	
	logHelper(fmt.Sprintf("[VPN] Configuring Full Tunnel via IF %s...", ifIndex))

	gwOut, _ := exec.Command("powershell", "-Command", "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1).NextHop").Output()
	currentGW := strings.TrimSpace(string(gwOut))

	if currentGW != "" && currentGW != "0.0.0.0" {
		logHelper(fmt.Sprintf("[VPN] Original Gateway: %s. Pinning server route.", currentGW))
		// Important: serverHost route MUST use the physical gateway
		exec.Command("route", "add", serverHost, currentGW, "metric", "1").Run()
	}

	logHelper("[VPN] Redirecting all traffic through TUN...")
	exec.Command("route", "add", "0.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	exec.Command("route", "add", "128.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	
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
