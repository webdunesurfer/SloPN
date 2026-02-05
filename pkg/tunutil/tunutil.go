package tunutil

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/songgao/water"
)

type Config struct {
	Name            string
	Addr            string
	Peer            string
	Mask            string
	MTU             int
	SkipSubnetRoute bool
}

func CreateInterface(cfg Config) (*water.Interface, error) {
	waterCfg := water.Config{
		DeviceType: water.TUN,
	}

	if runtime.GOOS == "linux" && cfg.Name != "" {
		waterCfg.PlatformSpecificParams = water.PlatformSpecificParams{
			Name: cfg.Name,
		}
	}

	ifce, err := water.New(waterCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v", err)
	}

	fmt.Printf("Created TUN interface: %s\n", ifce.Name())

	switch runtime.GOOS {
	case "darwin":
		return ifce, configureMacOS(ifce, cfg)
	case "linux":
		return ifce, configureLinux(ifce, cfg)
	default:
		return ifce, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func configureMacOS(ifce *water.Interface, cfg Config) error {
	fmt.Printf("Configuring %s: local=%s, peer=%s, netmask=%s, mtu=%d\n", ifce.Name(), cfg.Addr, cfg.Peer, cfg.Mask, cfg.MTU)
	
	// Command: ifconfig <name> <local> <peer> netmask <mask> mtu <mtu> up
	cmd := exec.Command("ifconfig", ifce.Name(), cfg.Addr, cfg.Peer, "netmask", cfg.Mask, "mtu", fmt.Sprintf("%d", cfg.MTU), "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig failed: %v (output: %s)", err, string(output))
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

	return nil
}

func configureLinux(ifce *water.Interface, cfg Config) error {
	addrCmd := exec.Command("ip", "addr", "add", cfg.Addr+"/24", "dev", ifce.Name())
	addrCmd.Run()

	exec.Command("ip", "link", "set", "dev", ifce.Name(), "mtu", fmt.Sprintf("%d", cfg.MTU)).Run()
	upCmd := exec.Command("ip", "link", "set", "dev", ifce.Name(), "up")
	if output, err := upCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ip link up failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Linux Interface %s ready: IP=%s/24\n", ifce.Name(), cfg.Addr)
	return nil
}
