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
	"strings"
	"sync"
	"time"

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

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		var i int
		fmt.Sscanf(value, "%d", &i)
		return i
	}
	return fallback
}

var (
	verbose   = flag.Bool("v", false, "Enable verbose logging")
	subnet    = flag.String("subnet", getEnv("SLOPN_SUBNET", "10.100.0.0/24"), "VPN Subnet")
	srvIP     = flag.String("ip", getEnv("SLOPN_IP", "10.100.0.1"), "Server Virtual IP")
	port      = flag.Int("port", 4242, "UDP Port to listen on")
	token     = flag.String("token", getEnv("SLOPN_TOKEN", "secret-token"), "Authentication token required for clients")
	enableNAT = flag.Bool("nat", false, "Enable NAT (MASQUERADE) for internet access")

	// Rate Limiting Config
	maxAttempts = flag.Int("max-attempts", getEnvInt("SLOPN_MAX_ATTEMPTS", 5), "Maximum failed attempts before ban")
	windowMins  = flag.Int("window", getEnvInt("SLOPN_WINDOW", 5), "Window in minutes for failed attempts")
	banMins     = flag.Int("ban-duration", getEnvInt("SLOPN_BAN_DURATION", 60), "Ban duration in minutes")
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time // IP -> List of failure timestamps
	banned   map[string]time.Time   // IP -> Ban expiration time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		banned:   make(map[string]time.Time),
	}
}

func (rl *RateLimiter) IsBanned(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	expiry, exists := rl.banned[ip]
	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		delete(rl.banned, ip)
		delete(rl.attempts, ip)
		return false
	}
	return true
}

func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.attempts[ip] = append(rl.attempts[ip], now)

	// Keep only attempts within the window
	window := time.Duration(*windowMins) * time.Minute
	var recent []time.Time
	for _, t := range rl.attempts[ip] {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}
	rl.attempts[ip] = recent

	if len(rl.attempts[ip]) >= *maxAttempts {
		rl.banned[ip] = now.Add(time.Duration(*banMins) * time.Minute)
		fmt.Printf("[SECURITY] IP %s BANNED for %d minutes due to too many failed attempts.\n", ip, *banMins)
	}
}

const ServerVersion = "0.5.7"

func main() {
	flag.Parse()

	sm, err := session.NewManager(*subnet, *srvIP)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	rl := NewRateLimiter()

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
			ifaceName := strings.TrimSpace(string(phyIfce))
			if ifaceName != "" {
				exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", *subnet, "-o", ifaceName, "-j", "MASQUERADE").Run()
				exec.Command("iptables", "-A", "FORWARD", "-i", "tun0", "-j", "ACCEPT").Run()
				exec.Command("iptables", "-A", "FORWARD", "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT").Run()
				fmt.Printf("NAT enabled on interface: %s\n", ifaceName)

				// DNS REDIRECTION:
				fmt.Println("Configuring DNS Redirection...")
				// Better way: CoreDNS is on the bridge. We redirect to the bridge gateway.
				gwOut, _ := exec.Command("sh", "-c", "ip route | grep default | awk '{print $3}'").Output()
				dockerGW := strings.TrimSpace(string(gwOut))
				if dockerGW != "" {
					exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-i", "tun0", "-p", "udp", "--dport", "53", "-j", "DNAT", "--to-destination", dockerGW).Run()
					exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-i", "tun0", "-p", "tcp", "--dport", "53", "-j", "DNAT", "--to-destination", dockerGW).Run()
					fmt.Printf("DNS queries from VPN will be redirected to Docker Gateway: %s\n", dockerGW)
				}
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
		go handleConnection(conn, ifce, sm, rl)
	}
}

func handleConnection(conn *quic.Conn, ifce *water.Interface, sm *session.Manager, rl *RateLimiter) {
	remoteIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	if rl.IsBanned(remoteIP) {
		fmt.Printf("[SECURITY] Refused connection from banned IP: %s\n", remoteIP)
		conn.CloseWithError(0x03, "banned")
		return
	}

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
		rl.RecordFailure(remoteIP)
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
