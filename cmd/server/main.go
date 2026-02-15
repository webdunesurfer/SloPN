// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"log"
	"math"
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
	"github.com/webdunesurfer/SloPN/pkg/obfuscator"
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
	verbose   = flag.Bool("v", true, "Enable verbose logging")
	subnet    = flag.String("subnet", getEnv("SLOPN_SUBNET", "10.100.0.0/24"), "VPN Subnet")
	srvIP     = flag.String("ip", getEnv("SLOPN_IP", "10.100.0.1"), "Server Virtual IP")
	port      = flag.Int("port", 4242, "UDP Port to listen on")
	token     = flag.String("token", getEnv("SLOPN_TOKEN", "secret-token"), "Authentication token required for clients")
	enableNAT = flag.Bool("nat", false, "Enable NAT (MASQUERADE) for internet access")
	obfs      = flag.Bool("obfs", true, "Enable protocol obfuscation (Reality-style)")
	mimic     = flag.String("mimic", getEnv("SLOPN_MIMIC", "www.google.com:443"), "Target server to mimic for unauthorized probes")
	diagMode  = flag.Bool("diag", false, "Enable diagnostic echo mode")
)

const ServerVersion = "0.9.5-diag-v15"

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
	window := time.Duration(5) * time.Minute
	var recent []time.Time
	for _, t := range rl.attempts[ip] {
		if now.Sub(t) < window {
			recent = append(recent, t)
		}
	}
	rl.attempts[ip] = recent

	if len(rl.attempts[ip]) >= 5 {
		rl.banned[ip] = now.Add(time.Duration(60) * time.Minute)
		logServer("BAN", "---", ip, fmt.Sprintf("Duration: 60m; Attempts: %d", len(rl.attempts[ip])))
	}
}

// Log formats: TIMESTAMP,EVENT,VIP,REMOTE_ADDR,DETAILS
func logServer(event, vip, remote, details string) {
	fmt.Printf("%s,%s,%s,%s,%s\n", time.Now().Format(time.RFC3339), event, vip, remote, details)
}

func main() {
	flag.Parse()

	sm, err := session.NewManager(*subnet, *srvIP)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	rl := NewRateLimiter()

	if runtime.GOOS == "linux" {
		if _, err := net.InterfaceByName("tun0"); err == nil {
			exec.Command("ip", "tuntap", "del", "mode", "tun", "name", "tun0").Run()
		}
		exec.Command("ip", "tuntap", "add", "mode", "tun", "name", "tun0", "nopi").Run()
	}

	tunCfg := tunutil.Config{
		Name: "tun0",
		Addr: sm.GetServerIP().String(),
		Peer: "10.100.0.2",
		Mask: "255.255.255.0",
		MTU:  1100,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatalf("Error creating TUN: %v", err)
	}
	defer ifce.Close()

	if runtime.GOOS == "linux" {
		exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		if *enableNAT {
			phyIfce, _ := exec.Command("sh", "-c", "ip route show default | awk '/default/ {print $5}'").Output()
			ifaceName := strings.TrimSpace(string(phyIfce))
			if ifaceName != "" {
				exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", *subnet, "-o", ifaceName, "-j", "MASQUERADE").Run()
				exec.Command("iptables", "-A", "FORWARD", "-i", "tun0", "-j", "ACCEPT").Run()
				exec.Command("iptables", "-A", "FORWARD", "-m", "state", "--state", "RELATED,ESTABLISHED", "-j", "ACCEPT").Run()
			}
		}
	}

	tlsConfig, err := certutil.GenerateSelfSignedConfig()
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	if *diagMode {
		fmt.Printf("DIAGNOSTIC MODE v14 ENABLED on :%d.\n", *port)
		mimicAddr, _ := net.ResolveUDPAddr("udp", *mimic)
		diagProxies := make(map[string]*net.UDPConn)
		var dpMu sync.Mutex

		for {
			buf := make([]byte, 2048)
			n, addr, err := udpConn.ReadFrom(buf)
			if err != nil { continue }

			ptype := "RAW"
			seq := "NONE"
			integrity := "N/A"

			if n > 0 && buf[0] == 0xFF {
				ptype = "PROBE"
				if n >= 16 {
					seq = string(buf[1:12])
					receivedCRC := binary.BigEndian.Uint32(buf[n-4:n])
					computedCRC := crc32.ChecksumIEEE(buf[:n-4])
					if receivedCRC == computedCRC { integrity = "OK" } else { integrity = "CORRUPT" }
				}
			} else if n > 0 && (buf[0]&0x80) != 0 {
				ptype = "QUIC-LONG"
			} else if n > 0 && (buf[0]&0x40) != 0 {
				ptype = "QUIC-SHORT"
			}

			counts := make(map[byte]int)
			for _, b := range buf[:n] { counts[b]++ }
			var entropy float64
			for _, count := range counts {
				p := float64(count) / float64(n)
				entropy -= p * math.Log2(p)
			}

			fmt.Printf("[DIAG] %-15v | Size: %4d | ID: %-10s | Int: %-7s | Type: %-10s | Ent: %4.2f\n", addr, n, seq, integrity, ptype, entropy)

			if ptype == "PROBE" {
				udpConn.WriteTo(buf[:n], addr)
			} else if mimicAddr != nil {
				remoteKey := addr.String()
				dpMu.Lock()
				proxyConn, exists := diagProxies[remoteKey]
				if !exists {
					proxyConn, _ = net.DialUDP("udp", nil, mimicAddr)
					diagProxies[remoteKey] = proxyConn
					go func(k string, c *net.UDPConn) {
						time.Sleep(30 * time.Second)
						dpMu.Lock()
						delete(diagProxies, k)
						dpMu.Unlock()
						c.Close()
					}(remoteKey, proxyConn)
					go func(clientAddr net.Addr, pc *net.UDPConn) {
						rBuf := make([]byte, 2048)
						for {
							pc.SetReadDeadline(time.Now().Add(5 * time.Second))
							rn, _ := pc.Read(rBuf)
							if rn <= 0 { return }
							udpConn.WriteTo(rBuf[:rn], clientAddr)
						}
					}(addr, proxyConn)
				}
				dpMu.Unlock()
				proxyConn.Write(buf[:n])
			}
		}
	}

	var finalConn net.PacketConn = udpConn
	if *obfs {
		fmt.Printf("Protocol Obfuscation (Reality) enabled. Mimicking: %s\n", *mimic)
		finalConn = obfuscator.NewRealityConn(udpConn, *token, *mimic)
	}

	listener, err := quic.Listen(finalConn, tlsConfig, &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

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
		conn.CloseWithError(0x03, "banned")
		return
	}
	stream, err := conn.AcceptStream(context.Background())
	if err != nil { return }
	defer stream.Close()

	var loginReq protocol.LoginRequest
	if err := json.NewDecoder(stream).Decode(&loginReq); err != nil { return }

	if loginReq.Token != *token {
		rl.RecordFailure(remoteIP)
		conn.CloseWithError(1, "unauthorized")
		return
	}

	vip, _ := sm.AllocateIP()
	resp := protocol.LoginResponse{
		Type: protocol.MessageTypeLoginResponse, Status: "success",
		AssignedVIP: vip.String(), ServerVIP: sm.GetServerIP().String(),
		ServerVersion: ServerVersion,
	}
	json.NewEncoder(stream).Encode(resp)
	sm.AddSession(vip, conn)

	ctx := conn.Context()
	go func() {
		defer sm.RemoveSession(vip.String())
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil { return }
			ifce.Write(iputil.AddHeader(data, runtime.GOOS == "linux"))
		}
	}()
	<-ctx.Done()
}
