//go:build darwin

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

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v", err)
	}

	fmt.Printf("Created TUN interface: %s\n", ifce.Name())

	cmd := exec.Command("ifconfig", ifce.Name(), cfg.Addr, cfg.Peer, "netmask", cfg.Mask, "mtu", fmt.Sprintf("%d", cfg.MTU), "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ifconfig failed: %v (output: %s)", err, string(output))
	}

	if cfg.NoRoute {
		fmt.Printf("macOS Interface %s ready (Skipped routing table modification)\n", ifce.Name())
		return ifce, nil
	}

	if cfg.SkipSubnetRoute {
		exec.Command("route", "delete", cfg.Peer).Run()
		routeCmd := exec.Command("route", "add", "-host", cfg.Peer, "-interface", ifce.Name())
		routeCmd.Run()
		fmt.Printf("macOS Interface %s ready (Host route to %s)\n", ifce.Name(), cfg.Peer)
	} else {
		exec.Command("route", "delete", "-net", "10.100.0.0/24").Run()
		routeCmd := exec.Command("route", "add", "-net", "10.100.0.0/24", "-interface", ifce.Name())
		routeCmd.Run()
		fmt.Printf("macOS Interface %s ready (Subnet route)\n", ifce.Name())
	}

	return ifce, nil
}