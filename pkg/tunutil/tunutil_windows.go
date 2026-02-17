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

	// List of common IDs for the TAP-Windows V9 driver
	ids := []string{"tap0901", "root\\tap0901"}
	var lastErr error

	for _, id := range ids {
		waterCfg := water.Config{
			DeviceType: water.TUN,
		}
		waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
			ComponentID:   id,
			InterfaceName: targetName,
			Network:       fmt.Sprintf("%s/%s", cfg.Addr, "24"),
		}

		ifce, err := water.New(waterCfg)
		if err == nil {
			fmt.Printf("Reusing existing TUN interface: %s (ID: %s)\n", targetName, id)
			return configureIP(ifce, targetName, cfg)
		}
		lastErr = err
	}

	// 2. If explicit open fails, try to find ANY available adapter with those IDs
	fmt.Printf("Interface '%s' not found or busy. Searching for available TAP adapter...\n", targetName)
	
	for _, id := range ids {
		waterCfg := water.Config{
			DeviceType: water.TUN,
		}
		waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
			ComponentID: id,
		}
		ifce, err := water.New(waterCfg)
		if err == nil {
			originalName := ifce.Name()
			if strings.EqualFold(originalName, targetName) {
				return configureIP(ifce, targetName, cfg)
			}

			fmt.Printf("Found available TAP interface: %s (ID: %s). Renaming to: %s\n", originalName, id, targetName)
			renameCmd := exec.Command("netsh", "interface", "set", "interface", "name="+originalName, "newname="+targetName)
			renameCmd.Run() // Ignore error, configureIP will catch if it failed
			
			return configureIP(ifce, targetName, cfg)
		}
		lastErr = err
	}

	return nil, fmt.Errorf("No TAP adapter found. Please ensure SloPN was installed with Administrator privileges and the TAP driver is not blocked by Antivirus/SecureBoot. (Last error: %v)", lastErr)
}

func configureIP(ifce *water.Interface, ifceName string, cfg Config) (*water.Interface, error) {
	// Command: netsh interface ip set address name="Name" static IP Mask
	// Pass arguments separately for proper quoting
	ipCmd := exec.Command("netsh", "interface", "ip", "set", "address", "name="+ifceName, "static", cfg.Addr, cfg.Mask)
	if output, err := ipCmd.CombinedOutput(); err != nil {
		ifce.Close()
		return nil, fmt.Errorf("netsh IP config failed for %s: %v (output: %s)", ifceName, err, string(output))
	}

	// Set MTU explicitly
	if cfg.MTU > 0 {
		mtuCmd := exec.Command("netsh", "interface", "ipv4", "set", "subinterface", ifceName, fmt.Sprintf("mtu=%d", cfg.MTU), "store=active")
		if output, err := mtuCmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: failed to set MTU for %s: %v (output: %s)\n", ifceName, err, string(output))
		}
	}

	fmt.Printf("Windows Interface %s ready: IP=%s MTU=%d\n", ifceName, cfg.Addr, cfg.MTU)
	return ifce, nil
}