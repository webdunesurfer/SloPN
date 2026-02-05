package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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

var (
	verbose = flag.Bool("v", false, "Enable verbose logging")
	subnet  = flag.String("subnet", "10.100.0.0/24", "VPN Subnet")
	srvIP   = flag.String("ip", "10.100.0.1", "Server Virtual IP")
	port    = flag.Int("port", 4242, "UDP Port to listen on")
)

func main() {
	flag.Parse()

	sm, err := session.NewManager(*subnet, *srvIP)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	tunCfg := tunutil.Config{
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

	// Linux system prep
	if runtime.GOOS == "linux" {
		exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
		exec.Command("sysctl", "-w", "net.ipv4.conf.all.rp_filter=0").Run()
		exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", ifce.Name())).Run()
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

	fmt.Printf("SloPN Server listening on :%d (Subnet: %s, VIP: %s)\n", *port, *subnet, sm.GetServerIP())

	// TUN -> QUIC loop
	go func() {
		packet := make([]byte, 2000)
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				return
			}

			destIP := iputil.GetDestinationIP(packet[:n])
			if destIP == nil || destIP.Equal(sm.GetServerIP()) {
				continue
			}

			if conn, ok := sm.GetSession(destIP.String()); ok {
				payload := iputil.StripHeader(packet[:n])
				err = conn.SendDatagram(payload)
				if err == nil && *verbose {
					fmt.Printf("Routed: %s\n", iputil.FormatPacketSummary(packet[:n]))
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

	vip, err := sm.AllocateIP()
	if err != nil {
		return
	}

	resp := protocol.LoginResponse{
		Type: protocol.MessageTypeLoginResponse, Status: "success",
		AssignedVIP: vip.String(), ServerVIP: sm.GetServerIP().String(),
	}
	json.NewEncoder(stream).Encode(resp)

	sm.AddSession(vip, conn)
	fmt.Printf("Client connected: %s -> %s\n", conn.RemoteAddr(), vip)

	ctx := conn.Context()
	isLinux := runtime.GOOS == "linux"
	go func() {
		defer sm.RemoveSession(vip.String())
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil {
				return
			}
			if *verbose {
				fmt.Printf("QUIC RECV [%s]: %s\n", vip, iputil.FormatPacketSummary(data))
			}
			payload := iputil.AddHeader(data, isLinux)
			ifce.Write(payload)
		}
	}()
	<-ctx.Done()
	fmt.Printf("Client disconnected: %s\n", vip)
}