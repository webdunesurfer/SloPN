//go:build windows

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
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
	exec.Command("netsh", "interface", "ip", "set", "dns", "name=slopn-tap0", "source=dhcp").Run()
	exec.Command("ipconfig", "/flushdns").Run()
}

// getLogs efficiently reads the last N bytes of the log file using native Go
func (h *Helper) getLogs() string {
	f, err := os.Open(LogPath)
	if err != nil {
		return "Unable to open log file"
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return ""
	}

	filesize := stat.Size()
	readSize := int64(2048) // Read last 2KB
	if filesize < readSize {
		readSize = filesize
	}

	buf := make([]byte, readSize)
	f.Seek(-readSize, 2) // Seek relative to end
	n, err := f.Read(buf)
	if err != nil {
		return ""
	}

	// cleanup potential partial line at start
	content := string(buf[:n])
	firstNewline := strings.Index(content, "\n")
	if firstNewline > -1 && firstNewline < len(content)-1 {
		return content[firstNewline+1:]
	}
	return content
}

func (h *Helper) getInterfaceIndex(name string) string {
	// Use netsh instead of PowerShell for performance
	// Output format: "Idx     Met         MTU          State                Name"
	out, err := exec.Command("netsh", "interface", "ip", "show", "interfaces").Output()
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, name) {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0] // The first field is the Index
			}
		}
	}
	return ""
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

	// Get Gateway using netsh or route print (avoiding PS)
	// Simplifying gateway detection for now: trust standard route command
	// Ideally we find the gateway IP, but for pinning route, we can try relying on existing routes
	
	// 1. Add host route to VPN server to prevent loops.
	// Since we are replacing PS, finding the exact Gateway IP natively is hard without parsing 'route print'.
	// Fallback: Let's use 'route add ...' without gateway IF we assume it picks the right interface,
	// BUT for safety, we simply assume the user has a default gateway 192.168.x.x or similar.
	// To fix the CPU issue quickly, I will skip the dynamic GW detection loop if it relies on PS 
	// and assume the OS handles the specific route to the server IP if we don't mess with 0.0.0.0/0 directly.
	// BUT we ARE adding 0.0.0.0/1.
	
	// Let's bring back a LIGHTWEIGHT gateway check using route print
	gwIP := getGatewayIP()
	if gwIP != "" {
		logHelper(fmt.Sprintf("[VPN] Pinning server route via %s", gwIP))
		exec.Command("route", "add", serverHost, "mask", "255.255.255.255", gwIP, "metric", "1").Run()
	}

	logHelper("[VPN] Redirecting all traffic through TUN...")
	exec.Command("route", "add", "0.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	exec.Command("route", "add", "128.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	
	h.setupDNS()
}

func getGatewayIP() string {
	out, _ := exec.Command("route", "print", "0.0.0.0").Output()
	// Look for: "          0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.50     25"
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			return fields[2] // Gateway IP
		}
	}
	return ""
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