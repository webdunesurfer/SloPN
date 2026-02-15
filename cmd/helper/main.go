// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/ipc"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/obfuscator"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

const (
	TCPAddr       = "127.0.0.1:54321"
	HelperVersion = "0.9.5-diag-v3"
)

type Helper struct {
	mu            sync.RWMutex
	state         string
	assignedVIP   string
	serverVIP     string
	serverAddr    string
	sni           string
	helperVersion string
	serverVersion string
	fullTunnel    bool
	obfuscate     bool
	verbose       bool
	bytesSent     uint64
	bytesRecv     uint64
	startTime     time.Time
	ipcSecret     string
	
	conn         *quic.Conn
	tunIfce      interface{}
	cancelVPN    context.CancelFunc
	vpnWG        sync.WaitGroup
}

func (h *Helper) logVerbose(msg string) {
	if h.verbose {
		logHelper("[VERBOSE] " + msg)
	}
}

func (h *Helper) loadIPCSecret() {
	data, err := os.ReadFile(SecretPath)
	if err != nil {
		if os.IsNotExist(err) {
			logHelper("IPC Secret not found. Generating new one...")
			b := make([]byte, 32)
			if _, err := rand.Read(b); err == nil {
				secret := hex.EncodeToString(b)
				// Save with 0644 so other users (GUI) can read it, but only System/Root can write
				if err := os.WriteFile(SecretPath, []byte(secret), 0644); err == nil {
					h.ipcSecret = secret
					logHelper("Generated and saved new IPC Secret.")
					return
				}
			}
		}
		logHelper(fmt.Sprintf("WARNING: Could not read IPC secret: %v. IPC will be unsecured!", err))
		return
	}
	h.ipcSecret = strings.TrimSpace(string(data))
	logHelper("IPC Secret loaded and security enabled.")
}

func (h *Helper) getStatus() ipc.Status {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return ipc.Status{
		State:         h.state,
		AssignedVIP:   h.assignedVIP,
		ServerVIP:     h.serverVIP,
		ServerAddr:    h.serverAddr,
		HelperVersion: HelperVersion,
		ServerVersion: h.serverVersion,
	}
}

func (h *Helper) getStats() ipc.Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	uptime := int64(0)
	if !h.startTime.IsZero() {
		uptime = int64(time.Since(h.startTime).Seconds())
	}
	return ipc.Stats{
		BytesSent: h.bytesSent,
		BytesRecv: h.bytesRecv,
		Uptime:    uptime,
	}
}

func logHelper(msg string) {
	f, _ := os.OpenFile(LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		fmt.Fprintf(f, "[%s] [v%s] %s\n", time.Now().Format("15:04:05"), HelperVersion, msg)
		f.Sync()
		f.Close()
	}
	fmt.Printf("[v%s] %s\n", HelperVersion, msg)
}

func (h *Helper) run(ctx context.Context) error {
	// Ensure configuration directory exists (for logs and secrets)
	os.MkdirAll(filepath.Dir(LogPath), 0755)

	// Check for verbose flag file (Windows-friendly debug method)
	flagPath := filepath.Join(filepath.Dir(LogPath), "verbose.flag")
	if _, err := os.Stat(flagPath); err == nil {
		h.verbose = true
	} else if !os.IsNotExist(err) {
		logHelper(fmt.Sprintf("[DEBUG] Verbose flag check error: %v (Path: %s)", err, flagPath))
	} else {
		// Just for deep debugging
		logHelper(fmt.Sprintf("[DEBUG] Verbose flag not found at: %s", flagPath))
	}

	logHelper(fmt.Sprintf("Helper starting. Verbose: %v, Args: %v", h.verbose, os.Args))
	h.loadIPCSecret()

	l, err := net.Listen("tcp", TCPAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer l.Close()
	
	logHelper(fmt.Sprintf("SUCCESS: Listening on %s", l.Addr().String()))

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go h.handleIPC(conn)
		}
	}()

	<-ctx.Done()
	logHelper("Stopping helper...")
	h.disconnect()
	h.vpnWG.Wait()
	return nil
}

func (h *Helper) handleIPC(c net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			logHelper(fmt.Sprintf("IPC Handler Panic: %v", r))
		}
		c.Close()
	}()

	var req ipc.Request
	decoder := json.NewDecoder(c)
	if err := decoder.Decode(&req); err != nil {
		return
	}

	var resp ipc.Response

	// Verify IPC Secret if enabled
	if h.ipcSecret != "" && req.IPCSecret != h.ipcSecret {
		logHelper(fmt.Sprintf("[SECURITY] Blocked unauthenticated IPC request from %s", c.RemoteAddr()))
		resp = ipc.Response{Status: "error", Message: "unauthorized: invalid IPC secret"}
		json.NewEncoder(c).Encode(resp)
		return
	}

	switch req.Command {
	case ipc.CmdConnect:
		logHelper(fmt.Sprintf("[IPC] Connecting to %s (SNI: %s, Obfs: %v)", req.ServerAddr, req.SNI, req.Obfuscate))
		err := h.connect(req.ServerAddr, req.Token, req.SNI, req.FullTunnel, req.Obfuscate)
		if err != nil {
			resp = ipc.Response{Status: "error", Message: err.Error()}
		} else {
			resp = ipc.Response{Status: "success", Message: "Connecting..."}
		}
	case ipc.CmdDisconnect:
		logHelper("[IPC] Disconnecting")
		h.disconnect()
		resp = ipc.Response{Status: "success", Message: "Disconnected"}
	case ipc.CmdGetStatus:
		resp = ipc.Response{Status: "success", Data: h.getStatus()}
	case ipc.CmdGetStats:
		resp = ipc.Response{Status: "success", Data: h.getStats()}
	case ipc.CmdGetLogs:
		resp = ipc.Response{Status: "success", Message: h.getLogs()}
	}

	json.NewEncoder(c).Encode(resp)
}

func (h *Helper) disconnect() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancelVPN != nil {
		h.cancelVPN()
		h.cancelVPN = nil
	}
	h.state = "disconnected"
	h.assignedVIP = ""
	h.serverVIP = ""
	h.serverVersion = ""
	h.conn = nil
	h.bytesSent = 0
	h.bytesRecv = 0
	h.startTime = time.Time{}
}

func (h *Helper) connect(addr, token, sni string, full, obfs bool) error {
	h.mu.Lock()
	if h.state == "connected" || h.state == "connecting" {
		h.mu.Unlock()
		return fmt.Errorf("already %s", h.state)
	}
	addr = strings.TrimSpace(addr)
	sni = strings.TrimSpace(sni)
	h.state = "connecting"
	h.serverAddr = addr
	h.sni = sni
	h.fullTunnel = full
	h.obfuscate = obfs
	h.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancelVPN = cancel
	h.mu.Unlock()

	go h.vpnLoop(ctx, addr, token, sni, full, obfs)
	return nil
}

func getLocalIP() string {
	targets := []string{"8.8.8.8:80", "1.1.1.1:80", "208.67.222.222:80"}
	for _, target := range targets {
		conn, err := net.DialTimeout("udp", target, 2*time.Second)
		if err == nil {
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			conn.Close()
			return localAddr.IP.String()
		}
	}
	return "0.0.0.0"
}

func (h *Helper) vpnLoop(ctx context.Context, addr, token, sni string, full, obfs bool) {
	h.vpnWG.Add(1)
	defer h.vpnWG.Done()
	
	addr = strings.TrimSpace(addr)
	sni = strings.TrimSpace(sni)
	
	logHelper(fmt.Sprintf("[VPN] Starting vpnLoop for %s (SNI: %s, Obfs: %v)", addr, sni, obfs))
	
	serverHost, _, _ := net.SplitHostPort(addr)
	var ifceName string

	defer func() {
		if r := recover(); r != nil {
			logHelper(fmt.Sprintf("[VPN] Loop Panic: %v", r))
		}

		if h.conn != nil {
			h.conn.CloseWithError(0, "logout")
		}
		
		h.cleanupRouting(full, serverHost, ifceName)
		h.disconnect()
		logHelper("[VPN] Loop exit complete.")
	}()

	h.setupRouting(full, serverHost, "", "") // serverVIP not known yet

	// Reality-style SNI Spoofing
	if sni == "" {
		sni = serverHost
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h3"},
		ServerName:         sni,
	}
	
	localIP := getLocalIP()
	logHelper(fmt.Sprintf("[VPN] Using local source IP: %s", localIP))
	udpConn, err := net.ListenPacket("udp4", localIP+":0")
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] UDP Listen error: %v", err))
		return
	}
	defer udpConn.Close()

	var finalConn net.PacketConn = udpConn
	if obfs {
		logHelper("[VPN] Protocol Obfuscation (Reality) enabled.")
		finalConn = obfuscator.NewRealityConn(udpConn, token, "") // Client doesn't need mimicTarget
	}
	
	remoteAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] Resolve error: %v", err))
		return
	}

	logHelper("[VPN] Dialing QUIC...")
	h.logVerbose("QUIC Config: KeepAlive=10s, Datagrams=true")
	
	dialCtx, dialCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dialCancel()

	conn, err := quic.Dial(dialCtx, finalConn, remoteAddr, tlsConf, &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 10 * time.Second,
	})
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] QUIC Dial error: %v", err))
		return
	}
	h.mu.Lock()
	h.conn = conn
	h.mu.Unlock()

	stream, err := conn.OpenStreamSync(dialCtx)
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] Stream error: %v", err))
		return
	}
	json.NewEncoder(stream).Encode(protocol.LoginRequest{
		Type: protocol.MessageTypeLoginRequest, Token: token,
		ClientVersion: HelperVersion, OS: runtime.GOOS,
	})

	var loginResp protocol.LoginResponse
	if err := json.NewDecoder(stream).Decode(&loginResp); err != nil {
		logHelper(fmt.Sprintf("[VPN] Login decode error: %v", err))
		return
	}
	stream.Close()

	if loginResp.Status != "success" {
		logHelper(fmt.Sprintf("[VPN] Login failed: %s", loginResp.Message))
		return
	}

	h.mu.Lock()
	h.state = "connected"
	h.assignedVIP = loginResp.AssignedVIP
	h.serverVIP = loginResp.ServerVIP
	h.serverVersion = loginResp.ServerVersion
	h.startTime = time.Now()
	h.mu.Unlock()

	logHelper(fmt.Sprintf("Connected! VIP: %s (Server v%s)", loginResp.AssignedVIP, loginResp.ServerVersion))

	tunCfg := tunutil.Config{
		Name: "slopn-tap0", // Use the name we established
		Addr: loginResp.AssignedVIP, Peer: loginResp.ServerVIP,
		Mask: "255.255.255.0", MTU: 1100,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] TUN error: %v", err))
		return
	}
	logHelper(fmt.Sprintf("[VPN] Interface %s created (IP: %s, MTU: %d)", tunCfg.Name, tunCfg.Addr, tunCfg.MTU))
	defer ifce.Close()
	ifceName = tunCfg.Name

	// Update routing with known serverVIP and dynamic IF Name
	h.tunIfce = ifce // Store for potential future use
	h.setupRouting(full, serverHost, loginResp.ServerVIP, ifceName)

	isLinux := runtime.GOOS == "linux"
	errChan := make(chan error, 2)

	// Log max datagram size
	h.logVerbose("MTU lowered to 900 for stability")

	// Stats ticker
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.mu.RLock()
				sent := h.bytesSent
				recv := h.bytesRecv
				h.mu.RUnlock()
				h.logVerbose(fmt.Sprintf("Heartbeat/Stats: Sent=%d bytes, Recv=%d bytes", sent, recv))
			}
		}
	}()

	go func() {
		first := true
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil {
				errChan <- err
				return
			}
			if first {
				h.logVerbose(fmt.Sprintf("First datagram received from server: %s", iputil.FormatPacketSummary(data)))
				first = false
			}
			h.mu.Lock()
			h.bytesRecv += uint64(len(data))
			h.mu.Unlock()
			ifce.Write(iputil.AddHeader(data, isLinux))
		}
	}()

	go func() {
		packet := make([]byte, 2000)
		first := true
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				errChan <- err
				return
			}
			payload := iputil.StripHeader(packet[:n])
			if first {
				h.logVerbose(fmt.Sprintf("First packet from TUN: %s", iputil.FormatPacketSummary(payload)))
				first = false
			}
			h.mu.Lock()
			h.bytesSent += uint64(len(payload))
			h.mu.Unlock()
			if err := conn.SendDatagram(payload); err != nil {
				h.logVerbose(fmt.Sprintf("SendDatagram error: %v (Size: %d)", err, len(payload)))
			}
		}
	}()

	select {
	case <-ctx.Done():
		logHelper("[VPN] Context cancelled")
		return
	case err := <-errChan:
		logHelper(fmt.Sprintf("[VPN] Data channel error: %v", err))
		return
	}
}
