// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/obfuscator"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

type Config struct {
	ServerAddr    string `json:"server_addr"`
	Token         string `json:"token"`
	SNI           string `json:"sni"`
	Verbose       bool   `json:"verbose"`
	HostRouteOnly bool   `json:"host_route_only"`
	NoRoute       bool   `json:"no_route"`
	FullTunnel    bool   `json:"full_tunnel"`
	Obfuscate     bool   `json:"obfuscate"`
}

var (
	configPath = flag.String("config", "config.json", "Path to config.json")
	overVerbose = flag.Bool("v", true, "Force verbose logging")
	fullTunnel = flag.Bool("full", false, "Enable Full Tunnel (route all traffic through VPN)")
)

func main() {
	flag.Parse()

	// 0. Load Config
	configFile, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Failed to open config: %v", err)
	}
	var cfg Config
	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		log.Fatalf("Failed to decode config: %v", err)
	}
	configFile.Close()

	// CLI flag can override config
	if *overVerbose {
		cfg.Verbose = true
	}
	if *fullTunnel {
		cfg.FullTunnel = true
	}

	// 1. Setup QUIC Client
	if cfg.SNI == "" {
		serverHost, _, _ := net.SplitHostPort(cfg.ServerAddr)
		cfg.SNI = serverHost
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
		ServerName:         cfg.SNI,
	}

	udpAddr, err := net.ResolveUDPAddr("udp", cfg.ServerAddr)
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenPacket("udp", "0.0.0.0:0")
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()

	var finalConn net.PacketConn = udpConn
	if cfg.Obfuscate {
		fmt.Printf("Protocol Obfuscation (Reality) enabled. SNI: %s\n", cfg.SNI)
		finalConn = obfuscator.NewRealityConn(udpConn, cfg.Token, "")
	}

	conn, err := quic.Dial(context.Background(), finalConn, udpAddr, tlsConf, &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "client exit")

	// 2. Authentication
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	json.NewEncoder(stream).Encode(protocol.LoginRequest{
		Type: protocol.MessageTypeLoginRequest, Token: cfg.Token,
		ClientVersion: "0.9.5-diag-v20", OS: runtime.GOOS,
	})

	var loginResp protocol.LoginResponse
	json.NewDecoder(stream).Decode(&loginResp)

	if loginResp.Status != "success" {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}

	fmt.Printf("Connected! Assigned VIP: %s (Server: %s)\n", loginResp.AssignedVIP, loginResp.ServerVIP)

	// 3. Setup TUN
	tunCfg := tunutil.Config{
		Addr: loginResp.AssignedVIP, Peer: loginResp.ServerVIP,
		Mask: "255.255.255.0", MTU: 1100,
		SkipSubnetRoute: cfg.HostRouteOnly,
		NoRoute:         cfg.NoRoute,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ifce.Close()

	// 4. Implement Full Tunnel Routing
	var currentGW string
	var serverHost string
	if cfg.FullTunnel && runtime.GOOS == "darwin" {
		fmt.Println("Configuring Full Tunnel (macOS)...")
		serverHost, _, _ = net.SplitHostPort(cfg.ServerAddr)
		
		// Get current gateway
		gwOut, _ := exec.Command("sh", "-c", "route -n get default | awk '/gateway: / {print $2}'").Output()
		currentGW = strings.TrimSpace(string(gwOut))
		if currentGW != "" {
			fmt.Printf("Current gateway: %s. Adding host route for server: %s\n", currentGW, serverHost)
			exec.Command("route", "add", "-host", serverHost, currentGW).Run()
		}

		exec.Command("route", "delete", "default").Run()
		exec.Command("route", "add", "default", loginResp.ServerVIP).Run()
		fmt.Println("Full Tunnel active. All traffic is now routed through SloPN.")

		// Ensure cleanup on exit
		defer func() {
			fmt.Println("\nRestoring original routing...")
			exec.Command("route", "delete", "default").Run()
			if currentGW != "" {
				exec.Command("route", "add", "default", currentGW).Run()
				exec.Command("route", "delete", "-host", serverHost).Run()
			}
		}()
	}

	isLinux := runtime.GOOS == "linux"
	
	// 5. Packet Forwarding
	
	// QUIC -> TUN
	go func() {
		for {
			data, err := conn.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			if cfg.Verbose {
				fmt.Printf("RECV: %s\n", iputil.FormatPacketSummary(data))
			}
			ifce.Write(iputil.AddHeader(data, isLinux))
		}
	}()

	// TUN -> QUIC
	go func() {
		packet := make([]byte, 2000)
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				return
			}
			if cfg.Verbose {
				fmt.Printf("SEND: %s\n", iputil.FormatPacketSummary(packet[:n]))
			}
			conn.SendDatagram(iputil.StripHeader(packet[:n]))
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("SloPN Client is running. Press Ctrl+C to stop.")
	<-sigChan
}