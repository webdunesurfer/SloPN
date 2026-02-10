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

	// Leave InterfaceName empty to let water pick the first available tap0901 device
	// This is much more robust than expecting a specific name like "slopn-tap0"
	waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
		ComponentID:   "tap0901",
		InterfaceName: "", 
		Network:       fmt.Sprintf("%s/%s", cfg.Addr, "24"),
	}

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v (make sure TAP-Windows driver is installed)", err)
	}

	// Use the actual name of the interface we got
	realName := ifce.Name()
	fmt.Printf("Created TUN interface: %s\n", realName)

	// For Windows, netsh is used to set the IP
	// netsh interface ip set address "InterfaceName" static IP Mask
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", realName, "static", cfg.Addr, cfg.Mask)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Close interface if config fails
		ifce.Close()
		return nil, fmt.Errorf("netsh failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Windows Interface %s ready: IP=%s\n", realName, cfg.Addr)
	return ifce, nil
}
