//go:build windows

package tunutil

import (
	"fmt"
	"os/exec"

	"github.com/songgao/water"
)

func CreateInterface(cfg Config) (*water.Interface, error) {
	waterCfg := water.Config{
		DeviceType: water.TUN,
	}

	waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
		ComponentID:   "tap0901",
		InterfaceName: cfg.Name,
		Network:       fmt.Sprintf("%s/%s", cfg.Addr, "24"), // Simplification for test
	}

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v (make sure TAP-Windows driver is installed and an adapter named '%s' exists)", err, cfg.Name)
	}

	fmt.Printf("Created TUN interface: %s\n", ifce.Name())

	// For Windows, netsh is used to set the IP
	// netsh interface ip set address "InterfaceName" static IP Mask
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", ifce.Name(), "static", cfg.Addr, cfg.Mask)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("netsh failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Windows Interface %s ready: IP=%s\n", ifce.Name(), cfg.Addr)
	return ifce, nil
}