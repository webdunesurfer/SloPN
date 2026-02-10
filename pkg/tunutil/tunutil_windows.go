//go:build windows

package tunutil

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/songgao/water"
)

func CreateInterface(cfg Config) (*water.Interface, error) {
	waterCfg := water.Config{
		DeviceType: water.TUN,
	}

	// 1. Initialize with standard ComponentID but NO specific name yet.
	// This allows water/driver to find or create the next available slot (e.g., "Local Area Connection X")
	waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
		ComponentID:   "tap0901",
		InterfaceName: "", 
		Network:       fmt.Sprintf("%s/%s", cfg.Addr, "24"),
	}

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v (driver issue?)", err)
	}

	originalName := ifce.Name()
	targetName := cfg.Name
	if targetName == "" {
		targetName = "slopn-tap0"
	}

	fmt.Printf("Created TUN interface: %s. Renaming to: %s\n", originalName, targetName)

	// 2. Explicitly RENAME the interface to what we want (slopn-tap0)
	// Command: netsh interface set interface name="OldName" newname="NewName"
	renameCmd := exec.Command("netsh", "interface", "set", "interface", fmt.Sprintf("name=\"%s\"", originalName), fmt.Sprintf("newname=\"%s\"", targetName))
	if output, err := renameCmd.CombinedOutput(); err != nil {
		// If rename fails, we log it but might continue if we can track the original name.
		// However, the user requested EXPLICIT naming.
		// Check if it's already named correctly (rare edge case)
		if !strings.EqualFold(originalName, targetName) {
			ifce.Close()
			return nil, fmt.Errorf("failed to rename interface from '%s' to '%s': %v (output: %s)", originalName, targetName, err, string(output))
		}
	}

	// 3. Configure IP using the NEW name
	// Command: netsh interface ip set address "NewName" static IP Mask
	ipCmd := exec.Command("netsh", "interface", "ip", "set", "address", fmt.Sprintf("name=\"%s\"", targetName), "static", cfg.Addr, cfg.Mask)
	if output, err := ipCmd.CombinedOutput(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("netsh IP config failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Windows Interface %s ready: IP=%s\n", targetName, cfg.Addr)
	
	// Note: We return the interface. The water.Interface struct still holds the internal handle,
	// but its Name() method might still return the old name depending on how water caches it.
	// For our Helper logic, we should probably update how we track the name.
	
	return ifce, nil
}