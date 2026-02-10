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

func (h *Helper) getAllActiveInterfaces() []string {
	// Get all interfaces that are connected
	out, err := exec.Command("netsh", "interface", "show", "interface").Output()
	if err != nil {
		return []string{}
	}

	var active []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Connected") {
			// Format: "Admin State    State          Type             Interface Name"
			// Example: "Enabled        Connected      Dedicated        Ethernet"
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				name := strings.Join(fields[3:], " ")
				active = append(active, name)
			}
		}
	}
	return active
}

func (h *Helper) setupDNS(ifceName string) {
	logHelper(fmt.Sprintf("[DNS] Configuring DNS for VPN interface %s...", ifceName))
	
	// 1. Force DNS on the VPN interface itself
	exec.Command("netsh", "interface", "ip", "set", "dns", fmt.Sprintf("name=\"%s\"", ifceName), "static", "10.100.0.1", "validate=no").Run()

	// 2. Aggressive Leak Protection: Force DNS on ALL other active interfaces to 10.100.0.1
	// This prevents Windows from using the ISP DNS via parallel queries.
	active := h.getAllActiveInterfaces()
	for _, name := range active {
		if strings.EqualFold(name, ifceName) {
			continue
		}
		logHelper(fmt.Sprintf("[DNS] Forcing protection on %s...", name))
		exec.Command("netsh", "interface", "ip", "set", "dns", fmt.Sprintf("name=\"%s\"", name), "static", "10.100.0.1", "validate=no").Run()
	}
	
	exec.Command("ipconfig", "/flushdns").Run()
	logHelper("[DNS] System-wide DNS protection active.")
}

func (h *Helper) restoreDNS(ifceName string) {
	logHelper("[DNS] Restoring system-wide DNS settings...")
	
	active := h.getAllActiveInterfaces()
	for _, name := range active {
		logHelper(fmt.Sprintf("[DNS] Restoring DHCP for %s...", name))
		exec.Command("netsh", "interface", "ip", "set", "dns", fmt.Sprintf("name=\"%s\"", name), "source=dhcp").Run()
	}
	
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
				return fields[0]
			}
		}
	}
	return ""
}

func (h *Helper) setupRouting(full bool, serverHost, serverVIP, ifceName string) {
	if serverVIP == "" {
		return
	}

	ifIndex := h.getInterfaceIndex(ifceName)
	if ifIndex == "" {
		logHelper(fmt.Sprintf("[VPN] Error: Could not find interface index for %s", ifceName))
		return
	}

	if !full {
		logHelper(fmt.Sprintf("[VPN] Adding split-tunnel route for 10.100.0.0/24 via %s (IF %s)", serverVIP, ifIndex))
		exec.Command("route", "add", "10.100.0.0", "mask", "255.255.255.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
		return
	}
	
	logHelper(fmt.Sprintf("[VPN] Configuring Full Tunnel via IF %s...", ifIndex))

	gwIP := getGatewayIP()
	if gwIP != "" {
		logHelper(fmt.Sprintf("[VPN] Pinning server route via %s", gwIP))
		exec.Command("route", "add", serverHost, "mask", "255.255.255.255", gwIP, "metric", "1").Run()
	}

	logHelper("[VPN] Redirecting all traffic through TUN...")
	exec.Command("route", "add", "0.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	exec.Command("route", "add", "128.0.0.0", "mask", "128.0.0.0", serverVIP, "IF", ifIndex, "metric", "1").Run()
	
	h.setupDNS(ifceName)
}

func getGatewayIP() string {
	out, _ := exec.Command("route", "print", "0.0.0.0").Output()
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

func (h *Helper) cleanupRouting(full bool, serverHost, ifceName string) {
	logHelper("[VPN] Cleaning up Windows routes...")
	
	if full {
		exec.Command("route", "delete", "0.0.0.0", "mask", "128.0.0.0").Run()
		exec.Command("route", "delete", "128.0.0.0", "mask", "128.0.0.0").Run()
		if serverHost != "" {
			exec.Command("route", "delete", serverHost).Run()
		}
		h.restoreDNS(ifceName)
	}
	
	exec.Command("route", "delete", "10.100.0.0", "mask", "255.255.255.0").Run()
}