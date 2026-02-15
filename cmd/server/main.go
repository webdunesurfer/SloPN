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
	if value, ok := os.LookupEnv(key); ok { return value }
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

const ServerVersion = "0.9.5-diag-v19"

func main() {
	flag.Parse()
	sm, _ := session.NewManager(*subnet, *srvIP)

	if runtime.GOOS == "linux" {
		exec.Command("ip", "tuntap", "del", "mode", "tun", "name", "tun0").Run()
		exec.Command("ip", "tuntap", "add", "mode", "tun", "name", "tun0", "nopi").Run()
	}

	tunCfg := tunutil.Config{Name: "tun0", Addr: sm.GetServerIP().String(), Peer: "10.100.0.2", Mask: "255.255.255.0", MTU: 1100}
	ifce, _ := tunutil.CreateInterface(tunCfg)
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

	tlsConfig, _ := certutil.GenerateSelfSignedConfig()
	udpConn, _ := net.ListenPacket("udp4", fmt.Sprintf("0.0.0.0:%d", *port))

	if *diagMode {
		fmt.Printf("DIAGNOSTIC MODE v19 ENABLED on :%d.\n", *port)
		mimicAddr, _ := net.ResolveUDPAddr("udp", *mimic)
		diagProxies := make(map[string]*net.UDPConn)
		var dpMu sync.Mutex
		lastSeenMap := make(map[string]time.Time)
		var lsMu sync.Mutex

		for {
			buf := make([]byte, 2048)
			n, addr, err := udpConn.ReadFrom(buf)
			if err != nil { continue }

			// INSTANT ECHO for probes to maintain sync
			if n > 0 && buf[0] == 0xFF {
				udpConn.WriteTo(buf[:n], addr)
			}

			// ASYNC LOGGING to prevent blocking the read loop
			go func(data []byte, clientAddr net.Addr) {
				remoteKey := clientAddr.String()
				lsMu.Lock()
				gap := time.Since(lastSeenMap[remoteKey])
				if lastSeenMap[remoteKey].IsZero() { gap = 0 }
				lastSeenMap[remoteKey] = time.Now()
				lsMu.Unlock()

				ptype := "RAW"
				seq := "NONE"
				integrity := "N/A"

				if data[0] == 0xFF {
					ptype = "PROBE"
					if len(data) >= 16 {
						seq = string(data[1:11])
						receivedCRC := binary.BigEndian.Uint32(data[len(data)-4:])
						computedCRC := crc32.ChecksumIEEE(data[:len(data)-4])
						if receivedCRC == computedCRC { integrity = "OK" } else { integrity = "CORRUPT" }
					}
				} else if (data[0]&0x80) != 0 {
					ptype = "QUIC-LONG"
				} else if (data[0]&0x40) != 0 {
					ptype = "QUIC-SHORT"
				}

				counts := make(map[byte]int)
				for _, b := range data { counts[b]++ }
				var entropy float64
				for _, count := range counts {
					p := float64(count) / float64(len(data))
					entropy -= p * math.Log2(p)
				}

				fmt.Printf("[DIAG] %-15v | Gap: %4dms | ID: %-10s | Int: %-7s | Type: %-10s | Ent: %4.2f\n", 
					clientAddr, gap.Milliseconds(), seq, integrity, ptype, entropy)

				// Mimic Proxy Handling
				if ptype != "PROBE" && mimicAddr != nil {
					dpMu.Lock()
					proxyConn, exists := diagProxies[remoteKey]
					if !exists {
						proxyConn, _ = net.DialUDP("udp", nil, mimicAddr)
						diagProxies[remoteKey] = proxyConn
						go func(k string, c *net.UDPConn) {
							time.Sleep(30 * time.Second)
							dpMu.Lock(); delete(diagProxies, k); dpMu.Unlock()
							c.Close()
						}(remoteKey, proxyConn)
						go func(ca net.Addr, pc *net.UDPConn) {
							rBuf := make([]byte, 2048)
							for {
								pc.SetReadDeadline(time.Now().Add(5 * time.Second))
								rn, _ := pc.Read(rBuf)
								if rn <= 0 { return }
								udpConn.WriteTo(rBuf[:rn], ca)
							}
						}(clientAddr, proxyConn)
					}
					dpMu.Unlock()
					proxyConn.Write(data)
				}
			}(buf[:n], addr)
		}
	}

	var finalConn net.PacketConn = udpConn
	if *obfs {
		finalConn = obfuscator.NewRealityConn(udpConn, *token, *mimic)
	}

	listener, _ := quic.Listen(finalConn, tlsConfig, &quic.Config{
		EnableDatagrams: true, KeepAlivePeriod: 10 * time.Second,
	})
	defer listener.Close()

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil { continue }
		go handleConnection(conn, ifce, sm)
	}
}

func handleConnection(conn *quic.Conn, ifce *water.Interface, sm *session.Manager) {
	stream, err := conn.AcceptStream(context.Background())
	if err != nil { return }
	defer stream.Close()

	var loginReq protocol.LoginRequest
	json.NewDecoder(stream).Decode(&loginReq)

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
