//go:build linux

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

	if cfg.Name != "" {
		waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
			Name: cfg.Name,
		}
	}

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v", err)
	}

	fmt.Printf("Created TUN interface: %s
", ifce.Name())

	addrCmd := exec.Command("ip", "addr", "add", cfg.Addr+"/24", "dev", ifce.Name())
	addrCmd.Run()

	exec.Command("ip", "link", "set", "dev", ifce.Name(), "mtu", fmt.Sprintf("%d", cfg.MTU)).Run()
	upCmd := exec.Command("ip", "link", "set", "dev", ifce.Name(), "up")
	if output, err := upCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ip link up failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Linux Interface %s ready: IP=%s/24
", ifce.Name(), cfg.Addr)
	return ifce, nil
}
