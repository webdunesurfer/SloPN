// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"github.com/webdunesurfer/SloPN/pkg/certutil"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/session"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var (
	verbose = flag.Bool("v", false, "Enable verbose logging")
	subnet  = flag.String("subnet", getEnv("SLOPN_SUBNET", "10.100.0.0/24"), "VPN Subnet")
	srvIP   = flag.String("ip", getEnv("SLOPN_IP", "10.100.0.1"), "Server Virtual IP")
	port    = flag.Int("port", 4242, "UDP Port to listen on")
	token   = flag.String("token", getEnv("SLOPN_TOKEN", "secret-token"), "Authentication token required for clients")
	enableNAT = flag.Bool("nat", false, "Enable NAT (MASQUERADE) for internet access")
)

const ServerVersion = "0.2.2"

func main() {
	flag.Parse()

	sm, err := session.NewManager(*subnet, *srvIP)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	if runtime.GOOS == "linux" {
		fmt.Println("Pre-creating TUN interface with nopi...")
		exec.Command("ip", "tuntap", "del", "mode", "tun", "name", "tun0").Run()
		err := exec.Command("ip", "tuntap", "add", "mode", "tun", "name", "tun0", "nopi").Run()
		if err != nil {
			fmt.Printf("Warning: failed to pre-create tun0: %v\n", err)
		}
	}

	tunCfg := tunutil.Config{
		Name: "tun0",
		Addr: sm.GetServerIP().String(),
		Peer: "10.100.0.2",
		Mask: "255.255.255.0",
		MTU:  1280,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatalf("Error creating TUN: %v", err)
	}
	defer ifce.Close()

	if runtime.GOOS == "linux" {
		exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		exec.Command("sysctl", "-w", "net.ipv4.conf.all.rp_filter=0").Run()
		exec.Command("sysctl", "-w", "net.ipv4.conf.default.rp_filter=0").Run()
		exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", ifce.Name())).Run()
		exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.accept_local=1", ifce.Name())).Run()

		if *enableNAT {
			fmt.Println("Enabling NAT (MASQUERADE)...")
			phyIfce, _ := exec.Command("sh", "-c", "ip route show default | awk '/default/ {print $5}'").Output()
			ifaceName := string(phyIfce)
			if ifaceName != "" {
				ifaceName = string(append([]byte(ifaceName[:len(ifaceName)-1]))) // trim newline
				exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", *subnet, "-o", ifaceName, "-j", "MASQUERADE").Run()
				exec.Command("iptables", "-A", "FORWARD", "-i", "tun0", "-j", "ACCEPT").Run()
				exec.Command("iptables", "-A", "FORWARD", "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT").Run()
				fmt.Printf("NAT enabled on interface: %s\n", ifaceName)
			}
		}
	}

	tlsConfig, err := certutil.GenerateSelfSignedConfig()
	if err != nil {
		log.Fatal(err)
	}

	listener, err := quic.ListenAddr(fmt.Sprintf("0.0.0.0:%d", *port), tlsConfig, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Printf("SloPN Server v%s listening on :%d (VIP: %s)\n", ServerVersion, *port, sm.GetServerIP())

	// TUN -> QUIC loop
	go func() {
		packet := make([]byte, 2000)
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				return
			}

			summary := iputil.FormatPacketSummary(packet[:n])
			destIP := iputil.GetDestinationIP(packet[:n])

			if conn, ok := sm.GetSession(destIP.String()); ok {
				if *verbose {
					fmt.Printf("TUN READ: %s\n", summary)
				}
				payload := iputil.StripHeader(packet[:n])
				err = conn.SendDatagram(payload)
				if err != nil && *verbose {
					log.Printf("QUIC Send error: %v", err)
				}
			}
		}
	}()

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			continue
		}
		go handleConnection(conn, ifce, sm)
	}
}

func handleConnection(conn *quic.Conn, ifce *water.Interface, sm *session.Manager) {
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		return
	}
	defer stream.Close()

	var loginReq protocol.LoginRequest
	if err := json.NewDecoder(stream).Decode(&loginReq); err != nil {
		return
	}

	// Validate Token
	if loginReq.Token != *token {
		remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		fmt.Printf("[AUTH_FAILURE] %s: invalid token\n", remoteIP)
		resp := protocol.LoginResponse{
			Type:          protocol.MessageTypeLoginResponse,
			Status:        "error",
			Message:       "Invalid authentication token",
			ServerVersion: ServerVersion,
		}
		json.NewEncoder(stream).Encode(resp)
		conn.CloseWithError(1, "unauthorized")
		return
	}

	vip, err := sm.AllocateIP()
	if err != nil {
		fmt.Printf("IP allocation failed for %v: %v\n", conn.RemoteAddr(), err)
		resp := protocol.LoginResponse{
			Type:          protocol.MessageTypeLoginResponse,
			Status:        "error",
			Message:       "Server failed to allocate IP",
			ServerVersion: ServerVersion,
		}
		json.NewEncoder(stream).Encode(resp)
		conn.CloseWithError(2, "ip allocation failed")
		return
	}

	resp := protocol.LoginResponse{
		Type: protocol.MessageTypeLoginResponse, Status: "success",
		AssignedVIP: vip.String(), ServerVIP: sm.GetServerIP().String(),
		ServerVersion: ServerVersion,
	}
	json.NewEncoder(stream).Encode(resp)

	sm.AddSession(vip, conn)
	fmt.Printf("Client connected: %s\n", vip)

	ctx := conn.Context()
	go func() {
		defer func() {
			sm.RemoveSession(vip.String())
			fmt.Printf("Client disconnected: %s\n", vip)
		}()
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil {
				return
			}
			// Only log data path in verbose mode
			if *verbose {
				fmt.Printf("QUIC RECV [%s]: %s\n", vip, iputil.FormatPacketSummary(data))
			}

			// OPTIMIZATION: Spoke-to-Spoke Fast Path
			// If destination is another client, route directly without TUN
			destIP := iputil.GetDestinationIP(data)
			if destIP != nil && !destIP.Equal(sm.GetServerIP()) {
				if targetConn, ok := sm.GetSession(destIP.String()); ok {
					if *verbose {
						fmt.Printf("  -> FAST-PATH: %s -> %s\n", vip, destIP)
					}
					targetConn.SendDatagram(data)
					continue
				}
			}

			// Always use false here because we pre-create tun0 with 'nopi'
			payload := iputil.AddHeader(data, false)
			_, err = ifce.Write(payload)
			if err != nil && *verbose {
				log.Printf("TUN Write error: %v (Hex: %s)", err, iputil.HexDump(payload))
			}
		}
	}()
	<-ctx.Done()
}