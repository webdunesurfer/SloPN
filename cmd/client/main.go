package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/quic-go/quic-go"
	"github.com/webdunesurfer/SloPN/pkg/iputil"
	"github.com/webdunesurfer/SloPN/pkg/protocol"
	"github.com/webdunesurfer/SloPN/pkg/tunutil"
)

type Config struct {
	ServerAddr string `json:"server_addr"`
	Token      string `json:"token"`
}

var (
	verbose    = flag.Bool("v", false, "Enable verbose logging")
	configPath = flag.String("config", "config.json", "Path to config.json")
	hostRoute  = flag.Bool("host-route", false, "Add host route only (for local multi-client testing)")
	noRoute    = flag.Bool("no-route", false, "Do not modify routing table (for manual testing)")
)

func main() {
	flag.Parse()

	configFile, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Failed to open config: %v", err)
	}
	var cfg Config
	json.NewDecoder(configFile).Decode(&cfg)
	configFile.Close()

	tlsConf := &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"slopn-protocol"}}
	conn, err := quic.DialAddr(context.Background(), cfg.ServerAddr, tlsConf, &quic.Config{EnableDatagrams: true})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.CloseWithError(0, "client exit")

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	json.NewEncoder(stream).Encode(protocol.LoginRequest{
		Type: protocol.MessageTypeLoginRequest, Token: cfg.Token,
		ClientVersion: "0.1.0", OS: runtime.GOOS,
	})

	var loginResp protocol.LoginResponse
	json.NewDecoder(stream).Decode(&loginResp)

	if loginResp.Status != "success" {
		log.Fatalf("Login failed: %s", loginResp.Message)
	}

	fmt.Printf("Connected! Assigned VIP: %s (Server: %s)\n", loginResp.AssignedVIP, loginResp.ServerVIP)

	tunCfg := tunutil.Config{
		Addr: loginResp.AssignedVIP, Peer: loginResp.ServerVIP,
		Mask: "255.255.255.0", MTU: 1280,
		SkipSubnetRoute: *hostRoute,
		NoRoute:         *noRoute,
	}
	ifce, err := tunutil.CreateInterface(tunCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer ifce.Close()

	isLinux := runtime.GOOS == "linux"
	// QUIC -> TUN
	go func() {
		for {
			data, err := conn.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			if *verbose {
				fmt.Printf("QUIC RECV: %s\n", iputil.FormatPacketSummary(data))
			}
			ifce.Write(iputil.AddHeader(data, isLinux))
		}
	}()

	// TUN -> QUIC
	packet := make([]byte, 2000)
	for {
		n, err := ifce.Read(packet)
		if err != nil {
			return
		}
		if *verbose {
			fmt.Printf("TUN READ: %s\n", iputil.FormatPacketSummary(packet[:n]))
		}
		conn.SendDatagram(iputil.StripHeader(packet[:n]))
	}
}