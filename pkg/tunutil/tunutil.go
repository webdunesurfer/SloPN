package tunutil

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/songgao/water"
)

type Config struct {
	Addr string
	Peer string
	Mask string
	MTU  int
}

func CreateInterface(cfg Config) (*water.Interface, error) {
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
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
	cmd := exec.Command("ifconfig", ifce.Name(), cfg.Addr, cfg.Peer, "netmask", cfg.Mask, "mtu", fmt.Sprintf("%d", cfg.MTU), "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ifconfig failed: %v (output: %s)", err, string(output))
	}

	exec.Command("route", "delete", "-net", "10.100.0.0/24").Run()
	routeCmd := exec.Command("route", "add", "-net", "10.100.0.0/24", "-interface", ifce.Name())
	if output, err := routeCmd.CombinedOutput(); err != nil {
		fmt.Printf("Route add warning: %v (output: %s)\n", err, string(output))
	}

	fmt.Printf("macOS Interface %s ready: Local=%s, Peer=%s\n", ifce.Name(), cfg.Addr, cfg.Peer)
	return nil
}

func configureLinux(ifce *water.Interface, cfg Config) error {
	// 1. Set IP address
	addrCmd := exec.Command("ip", "addr", "add", cfg.Addr+"/24", "dev", ifce.Name())
	if output, err := addrCmd.CombinedOutput(); err != nil {
		fmt.Printf("Note: IP assignment warning: %v (output: %s)\n", err, string(output))
	}

	// 2. Set MTU
	exec.Command("ip", "link", "set", "dev", ifce.Name(), "mtu", fmt.Sprintf("%d", cfg.MTU)).Run()

	// 3. Bring interface up
	upCmd := exec.Command("ip", "link", "set", "dev", ifce.Name(), "up")
	if output, err := upCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ip link up failed: %v (output: %s)", err, string(output))
	}

	fmt.Printf("Linux Interface %s ready: IP=%s/24\n", ifce.Name(), cfg.Addr)
	return nil
}
