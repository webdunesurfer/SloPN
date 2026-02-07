// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/ipc"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

const (
	TCPAddr       = "0.0.0.0:54321"
	HelperVersion = "0.1.3"
)

type Helper struct {
	mu            sync.RWMutex
	state         string
	assignedVIP   string
	serverVIP     string
	serverAddr    string
	helperVersion string
	serverVersion string
	fullTunnel    bool
	bytesSent     uint64
	bytesRecv     uint64
	startTime     time.Time
	
	conn         *quic.Conn
	tunIfce      interface{}
	cancelVPN    context.CancelFunc
	vpnWG        sync.WaitGroup
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

func (h *Helper) getLogs() string {
	out, err := exec.Command("tail", "-n", "100", "helper.log").Output()
	if err != nil {
		return "Failed to read logs: " + err.Error()
	}
	return string(out)
}

func logHelper(msg string) {
	f, _ := os.OpenFile("helper.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		fmt.Fprintf(f, "[%s] [v%s] %s\n", time.Now().Format("15:04:05"), HelperVersion, msg)
		f.Sync()
		f.Close()
	}
	fmt.Printf("[v%s] %s\n", HelperVersion, msg)
}

func main() {
	logHelper(fmt.Sprintf("SloPN Helper Starting. PID: %d", os.Getpid()))
	
	defer func() {
		if r := recover(); r != nil {
			logHelper(fmt.Sprintf("CRITICAL PANIC: %v", r))
		}
	}()

	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		log.Fatal("Unsupported OS")
	}

	h := &Helper{state: "disconnected"}

	l, err := net.Listen("tcp", TCPAddr)
	if err != nil {
		logHelper(fmt.Sprintf("CRITICAL: Failed to listen: %v", err))
		os.Exit(1)
	}
	
	logHelper(fmt.Sprintf("SUCCESS: Listening on %s", l.Addr().String()))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logHelper("Shutdown signal received. Cleaning up...")
		h.disconnect()
		h.vpnWG.Wait()
		l.Close()
		logHelper("Helper exited gracefully.")
		os.Exit(0)
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go h.handleIPC(conn)
	}
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
	switch req.Command {
	case ipc.CmdConnect:
		logHelper(fmt.Sprintf("[IPC] Connecting to %s", req.ServerAddr))
		err := h.connect(req.ServerAddr, req.Token, req.FullTunnel)
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
	h.startTime = time.Time{}
}

func (h *Helper) connect(addr, token string, full bool) error {
	h.mu.Lock()
	if h.state == "connected" || h.state == "connecting" {
		h.mu.Unlock()
		return fmt.Errorf("already %s", h.state)
	}
	h.state = "connecting"
	h.serverAddr = addr
	h.fullTunnel = full
	h.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.cancelVPN = cancel
	h.mu.Unlock()

	go h.vpnLoop(ctx, addr, token, full)
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

func (h *Helper) vpnLoop(ctx context.Context, addr, token string, full bool) {
	h.vpnWG.Add(1)
	defer h.vpnWG.Done()
	
	logHelper(fmt.Sprintf("[VPN] Starting vpnLoop for %s", addr))
	
	serverHost, _, _ := net.SplitHostPort(addr)
	var currentGW string

	defer func() {
		if r := recover(); r != nil {
			logHelper(fmt.Sprintf("[VPN] Loop Panic: %v", r))
		}

		if h.conn != nil {
			h.conn.CloseWithError(0, "logout")
		}
		
		logHelper("[VPN] Cleaning up routing...")
		if full && runtime.GOOS == "darwin" {
			exec.Command("route", "delete", "default").Run()
			if currentGW != "" {
				exec.Command("route", "add", "default", currentGW).Run()
				logHelper(fmt.Sprintf("[VPN] Restored default GW: %s", currentGW))
			}
			exec.Command("route", "delete", "-host", serverHost).Run()
			logHelper(fmt.Sprintf("[VPN] Removed host route for: %s", serverHost))
		}
		
		h.disconnect()
		logHelper("[VPN] Loop exit complete.")
	}()

	if runtime.GOOS == "darwin" {
		gwOut, _ := exec.Command("sh", "-c", "route -n get default | awk '/gateway: / {print $2}'").Output()
		currentGW = strings.TrimSpace(string(gwOut))
		if currentGW != "" {
			logHelper(fmt.Sprintf("[VPN] Found gateway: %s. Adding host route for %s", currentGW, serverHost))
			exec.Command("route", "add", "-host", serverHost, currentGW).Run()
		}
	}

	tlsConf := &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"slopn-protocol"}}
	
	localIP := getLocalIP()
	logHelper(fmt.Sprintf("[VPN] Using local source IP: %s", localIP))
	udpConn, err := net.ListenPacket("udp4", localIP+":0")
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] UDP Listen error: %v", err))
		return
	}
	defer udpConn.Close()
	
	remoteAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] Resolve error: %v", err))
		return
	}

	logHelper("[VPN] Dialing QUIC...")
	
	dialCtx, dialCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dialCancel()

	conn, err := quic.Dial(dialCtx, udpConn, remoteAddr, tlsConf, &quic.Config{EnableDatagrams: true})
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
		Addr: loginResp.AssignedVIP, Peer: loginResp.ServerVIP,
		Mask: "255.255.255.0", MTU: 1280,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		logHelper(fmt.Sprintf("[VPN] TUN error: %v", err))
		return
	}
	defer ifce.Close()

	if full && runtime.GOOS == "darwin" {
		logHelper("[VPN] Configuring Full Tunnel (v4 + v6 protection)...")
		exec.Command("route", "delete", "default").Run()
		exec.Command("route", "delete", "-inet6", "default").Run()
		exec.Command("route", "add", "default", loginResp.ServerVIP).Run()
		logHelper("[VPN] Routing table updated.")
	}

	isLinux := runtime.GOOS == "linux"
	errChan := make(chan error, 2)

	go func() {
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil {
				errChan <- err
				return
			}
			h.mu.Lock()
			h.bytesRecv += uint64(len(data))
			h.mu.Unlock()
			ifce.Write(iputil.AddHeader(data, isLinux))
		}
	}()

	go func() {
		packet := make([]byte, 2000)
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				errChan <- err
				return
			}
			payload := iputil.StripHeader(packet[:n])
			h.mu.Lock()
			h.bytesSent += uint64(len(payload))
			h.mu.Unlock()
			conn.SendDatagram(payload)
		}
	}()

	select {
	case <-ctx.Done():
		logHelper("[VPN] Context cancelled")
		return
	case <-errChan:
		logHelper("[VPN] Data channel error")
		return
	}
}
