package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"github.com/webdunesurfer/SloPN/pkg/certutil"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/session"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

func main() {
	// 1. Setup Session Manager (IPAM)
	sm, err := session.NewManager("10.100.0.0/24", "10.100.0.1")
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}

	// 2. Setup TUN Interface
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

	// Ensure IP forwarding is on
	exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	exec.Command("sysctl", "-w", "net.ipv4.conf.all.rp_filter=0").Run()

	// 3. Setup QUIC Server
	tlsConfig, err := certutil.GenerateSelfSignedConfig()
	if err != nil {
		log.Fatal(err)
	}

	listener, err := quic.ListenAddr("0.0.0.0:4242", tlsConfig, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Printf("SloPN Server listening on :4242 (Virtual IP: %s)\n", sm.GetServerIP())

	// 4. Packet Routing Loop (TUN -> QUIC)
	go func() {
		fmt.Println("Starting TUN -> QUIC routing loop...")
		packet := make([]byte, 2000)
		for {
			n, err := ifce.Read(packet)
			if err != nil {
				log.Printf("TUN Read error: %v", err)
				return
			}

			destIP := iputil.GetDestinationIP(packet[:n])
			if destIP == nil {
				continue
			}

			// DEBUG: log every packet read from TUN
			// fmt.Printf("TUN read: %d bytes for %s\n", n, destIP)

			if destIP.Equal(sm.GetServerIP()) {
				continue
			}

			// Route to the correct client session
			if conn, ok := sm.GetSession(destIP.String()); ok {
				err = conn.SendDatagram(packet[:n])
				if err != nil {
					log.Printf("QUIC Send error to %s: %v", destIP, err)
				} else {
					fmt.Printf("Routed %d bytes to %s\n", n, destIP)
				}
			}
		}
	}()

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn, ifce, sm)
	}
}

func handleConnection(conn *quic.Conn, ifce *water.Interface, sm *session.Manager) {
	fmt.Printf("New connection from %v\n", conn.RemoteAddr())

	// Step 1: Authentication & IP Allocation
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Printf("Failed to accept control stream: %v", err)
		return
	}
	defer stream.Close()

	decoder := json.NewDecoder(stream)
	var loginReq protocol.LoginRequest
	if err := decoder.Decode(&loginReq); err != nil {
		log.Printf("Handshake decode error: %v", err)
		return
	}

	// Allocate a Virtual IP
	vip, err := sm.AllocateIP()
	if err != nil {
		log.Printf("IP Allocation failed: %v", err)
		return
	}

	resp := protocol.LoginResponse{
		Type:        protocol.MessageTypeLoginResponse,
		Status:      "success",
		AssignedVIP: vip.String(),
		ServerVIP:   sm.GetServerIP().String(),
		Message:     fmt.Sprintf("Welcome to SloPN. Your IP is %s", vip),
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("Handshake encode error: %v", err)
		return
	}

	// Register the session
	sm.AddSession(vip, conn)
	
	// Use a context to handle session cleanup when connection closes
	fmt.Printf("Client %s authenticated. VIP assigned: %s\n", conn.RemoteAddr(), vip)

	// Step 2: Receive loop (QUIC -> TUN)
	ctx := conn.Context()
	go func() {
		defer sm.RemoveSession(vip.String())
		for {
			data, err := conn.ReceiveDatagram(ctx)
			if err != nil {
				log.Printf("QUIC Receive error from %s: %v", vip, err)
				return
			}
			// fmt.Printf("QUIC -> TUN: %d bytes from %s\n", len(data), vip)
			_, err = ifce.Write(data)
			if err != nil {
				log.Printf("TUN Write error: %v", err)
				return
			}
		}
	}()
	
	// Keep handleConnection alive as long as the QUIC connection is active
	<-ctx.Done()
	fmt.Printf("Session closed for %s (%s)\n", conn.RemoteAddr(), vip)
}