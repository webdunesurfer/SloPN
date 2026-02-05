package tunutil

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

// Config holds the configuration for the TUN interface
type Config struct {
	Addr string // Local VIP, e.g., "10.100.0.1"
	Peer string // Peer VIP, e.g., "10.100.0.2"
	Mask string // e.g., "255.255.255.0"
	MTU  int    // e.g., 1280 (per ADR)
}

// CreateInterface creates and configures a TUN interface
func CreateInterface(cfg Config) (*water.Interface, error) {
	// 1. Create the interface
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v", err)
	}

	fmt.Printf("Created TUN interface: %s\n", ifce.Name())

	// 2. Configure the interface (macOS specific utun)
	cmd := exec.Command("ifconfig", ifce.Name(), cfg.Addr, cfg.Peer, "netmask", cfg.Mask, "mtu", fmt.Sprintf("%d", cfg.MTU), "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to configure interface %s: %v (output: %s)", ifce.Name(), err, string(output))
	}

	fmt.Printf("Interface %s configured: Local=%s, Peer=%s\n", ifce.Name(), cfg.Addr, cfg.Peer)

	// 3. Add explicit route for the subnet
	routeCmd := exec.Command("route", "add", "-net", "10.100.0.0/24", cfg.Addr)
	if output, err := routeCmd.CombinedOutput(); err != nil {
		fmt.Printf("Note: route add note: %v (output: %s)\n", err, string(output))
	}

	return ifce, nil
}
