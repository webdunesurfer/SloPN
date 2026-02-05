package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"github.com/webdunesurfer/SloPN/pkg/certutil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

func main() {
	// 1. Setup TUN Interface
	tunCfg := tunutil.Config{
		Addr: "10.100.0.1",
		Mask: "255.255.255.0",
		MTU:  1280, // Per ADR
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatalf("Error creating TUN: %v. (Note: May require sudo)", err)
	}
	defer ifce.Close()

	// 2. Setup QUIC Server
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

	fmt.Println("SloPN Server listening on :4242")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn, ifce)
	}
}

func handleConnection(conn *quic.Conn, ifce *water.Interface) {
	defer conn.CloseWithError(0, "connection closed")

	fmt.Printf("New connection from %v\n", conn.RemoteAddr())

	// Step 1: Authentication via Control Stream
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

	// For Phase 2, we still use static mapping for testing point-to-point
	resp := protocol.LoginResponse{
		Type:        protocol.MessageTypeLoginResponse,
		Status:      "success",
		AssignedVIP: "10.100.0.2",
		ServerVIP:   "10.100.0.1",
		Message:     "Phase 2 Tunnel Ready",
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("Handshake encode error: %v", err)
		return
	}

	fmt.Println("Client authenticated. Starting packet loop.")

	// Step 2: Packet Forwarding Loops
	
	// QUIC -> TUN
	go func() {
		for {
			data, err := conn.ReceiveDatagram(context.Background())
			if err != nil {
				log.Printf("QUIC Receive error: %v", err)
				return
			}
			_, err = ifce.Write(data)
			if err != nil {
				log.Printf("TUN Write error: %v", err)
				return
			}
		}
	}()

	// TUN -> QUIC
	// NOTE: In Phase 2 Point-to-Point, we just forward everything from TUN to the only connected client.
	// Phase 3 will implement proper routing.
	packet := make([]byte, 1500)
	for {
		n, err := ifce.Read(packet)
		if err != nil {
			log.Printf("TUN Read error: %v", err)
			return
		}
		
		err = conn.SendDatagram(packet[:n])
		if err != nil {
			log.Printf("QUIC Send error: %v", err)
			return
		}
	}
}