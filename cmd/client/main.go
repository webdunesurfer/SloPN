package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

func main() {
	// 1. Setup QUIC Client
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"slopn-protocol"},
	}

	conn, err := quic.DialAddr(context.Background(), "localhost:4242", tlsConf, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "client exit")

	fmt.Println("Connected to server")

	// 2. Authentication via Control Stream
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	loginReq := protocol.LoginRequest{
		Type:          protocol.MessageTypeLoginRequest,
		Token:         "test-token",
		ClientVersion: "0.1.0",
		OS:            "macos",
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(loginReq); err != nil {
		log.Fatal(err)
	}

	decoder := json.NewDecoder(stream)
	var loginResp protocol.LoginResponse
	if err := decoder.Decode(&loginResp); err != nil {
		log.Fatal(err)
	}

	if loginResp.Status != "success" {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}

	fmt.Printf("Login successful. Assigned VIP: %s\n", loginResp.AssignedVIP)

	// 3. Setup TUN Interface with assigned IP
	tunCfg := tunutil.Config{
		Addr: loginResp.AssignedVIP,
		Peer: loginResp.ServerVIP,
		Mask: "255.255.255.0",
		MTU:  1280, // Per ADR
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatalf("Error creating TUN: %v. (Note: May require sudo)", err)
	}
	defer ifce.Close()

	// 4. Packet Forwarding Loops
	
	// QUIC -> TUN
	go func() {
		for {
			data, err := conn.ReceiveDatagram(context.Background())
			if err != nil {
				log.Printf("QUIC Receive error: %v", err)
				return
			}
			fmt.Printf("QUIC -> TUN: %d bytes\n", len(data))
			_, err = ifce.Write(data)
			if err != nil {
				log.Printf("TUN Write error: %v", err)
				return
			}
		}
	}()

	// TUN -> QUIC
	packet := make([]byte, 2000)
	for {
		n, err := ifce.Read(packet)
		if err != nil {
			log.Printf("TUN Read error: %v", err)
			return
		}
		
		fmt.Printf("TUN -> QUIC: %d bytes\n", n)
		err = conn.SendDatagram(packet[:n])
		if err != nil {
			log.Printf("QUIC Send error: %v", err)
			return
		}
	}
}
