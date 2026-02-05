package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/certutil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
)

func main() {
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
		go handleConnection(conn)
	}
}

func handleConnection(conn *quic.Conn) {
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

	fmt.Printf("Login request from %s (OS: %s, Version: %s)\n", loginReq.Token, loginReq.OS, loginReq.ClientVersion)

	// For Phase 1, any token is fine
	resp := protocol.LoginResponse{
		Type:        protocol.MessageTypeLoginResponse,
		Status:      "success",
		AssignedVIP: "10.100.0.2",
		ServerVIP:   "10.100.0.1",
		Message:     "Welcome to Phase 1",
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("Handshake encode error: %v", err)
		return
	}

	fmt.Println("Client authenticated successfully")

	// Step 2: Receive Datagrams
	for {
		data, err := conn.ReceiveDatagram(context.Background())
		if err != nil {
			log.Printf("ReceiveDatagram error: %v", err)
			return
		}
		fmt.Printf("Received datagram (%d bytes): %s\n", len(data), string(data))
	}
}