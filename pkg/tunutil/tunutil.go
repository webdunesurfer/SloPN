package tunutil

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

// Config holds the configuration for the TUN interface
type Config struct {
	Addr string // e.g., "10.100.0.1"
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

	// 2. Configure the interface (macOS specific)
	// Command: ifconfig <name> <addr> <dest_addr> netmask <mask> mtu <mtu> up
	cmd := exec.Command("ifconfig", ifce.Name(), cfg.Addr, cfg.Addr, "netmask", cfg.Mask, "mtu", fmt.Sprintf("%d", cfg.MTU), "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to configure interface %s: %v (output: %s)", ifce.Name(), err, string(output))
	}

	fmt.Printf("Interface %s configured with IP %s and MTU %d\n", ifce.Name(), cfg.Addr, cfg.MTU)

	return ifce, nil
}