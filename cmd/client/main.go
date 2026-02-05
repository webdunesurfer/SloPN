package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
)

func main() {
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

	// Step 1: Authentication via Control Stream
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

	// Step 2: Send dummy datagrams
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		msg := fmt.Sprintf("Ping from client at %s", time.Now().Format(time.Kitchen))
		err := conn.SendDatagram([]byte(msg))
		if err != nil {
			log.Printf("SendDatagram error: %v", err)
			return
		}
		fmt.Printf("Sent datagram: %s\n", msg)
	}
}