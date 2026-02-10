//go:build windows

package tunutil

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

func CreateInterface(cfg Config) (*water.Interface, error) {
	targetName := cfg.Name
	if targetName == "" {
		targetName = "slopn-tap0"
	}

	// 1. First, try to open the interface with the explicit name "slopn-tap0"
	// This works if the interface was already created and renamed by a previous run.
	waterCfg := water.Config{
		DeviceType: water.TUN,
	}
	waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
		ComponentID:   "tap0901",
		InterfaceName: targetName, 
		Network:       fmt.Sprintf("%s/%s", cfg.Addr, "24"),
	}

	ifce, err := water.New(waterCfg)
	if err == nil {
		fmt.Printf("Reusing existing TUN interface: %s\n", targetName)
		return configureIP(ifce, targetName, cfg)
	}

	// 2. If explicit open fails, try to find ANY available tap0901 and rename it.
	fmt.Printf("Interface '%s' not found or busy. Searching for available TAP adapter...\n", targetName)
	
	waterCfg.PlatformSpecificParams.InterfaceName = ""
	ifce, err = water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create/find any TAP interface: %v", err)
	}

	originalName := ifce.Name()
	fmt.Printf("Found available TAP interface: %s. Renaming to: %s\n", originalName, targetName)

	// 3. Rename the generic interface to our standard name
	renameCmd := exec.Command("netsh", "interface", "set", "interface", fmt.Sprintf("name=\"%s\"", originalName), fmt.Sprintf("newname=\"%s\"", targetName))
	if output, err := renameCmd.CombinedOutput(); err != nil {
		if !strings.Contains(strings.ToLower(string(output)), "already exists") {
			ifce.Close()
			return nil, fmt.Errorf("failed to rename interface to '%s': %v (output: %s)", targetName, err, string(output))
		}
		fmt.Printf("Warning: Interface '%s' already exists, might be a naming collision.\n", targetName)
	}

	return configureIP(ifce, targetName, cfg)
}

func configureIP(ifce *water.Interface, ifceName string, cfg Config) (*water.Interface, error) {
	// Command: netsh interface ip set address "Name" static IP Mask
	ipCmd := exec.Command("netsh", "interface", "ip", "set", "address", fmt.Sprintf("name=\"%s\"", ifceName), "static", cfg.Addr, cfg.Mask)
	if output, err := ipCmd.CombinedOutput(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("netsh IP config failed for %s: %v (output: %s)", ifceName, err, string(output))
	}

	fmt.Printf("Windows Interface %s ready: IP=%s\n", ifceName, cfg.Addr)
	return ifce, nil
}
