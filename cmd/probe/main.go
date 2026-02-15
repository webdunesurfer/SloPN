package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	target  = flag.String("addr", "", "Server address (e.g. 1.2.3.4:4242)")
)

func logDiag(msg string) {
	fmt.Printf("[%s] [PROBE] %s\n", time.Now().Format("15:04:05"), msg)
}

func testUDP(addr string) {
	logDiag("--- TEST A: Raw UDP Baseline ---")
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		logDiag(fmt.Sprintf("Resolve failed: %v", err))
		return
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		logDiag(fmt.Sprintf("UDP Dial failed: %v", err))
		return
	}
	defer conn.Close()

	payload := []byte("SloPN-Diagnostic-Ping")
	start := time.Now()
	conn.Write(payload)

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		logDiag(fmt.Sprintf("Baseline FAILED (No echo): %v", err))
	} else {
		logDiag(fmt.Sprintf("Baseline SUCCESS: Received %d bytes in %v", n, time.Since(start)))
	}
}

func testMTU(addr string) {
	logDiag("--- TEST B: MTU Sweep ---")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	sizes := []int{500, 800, 1000, 1200, 1400}

	for _, sz := range sizes {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		payload := make([]byte, sz)
		copy(payload, []byte(fmt.Sprintf("MTU-TEST-%d", sz)))
		
		start := time.Now()
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			logDiag(fmt.Sprintf("MTU %d: FAILED", sz))
		} else {
			logDiag(fmt.Sprintf("MTU %d: SUCCESS (%v)", sz, time.Since(start)))
		}
		conn.Close()
		time.Sleep(200 * time.Millisecond)
	}
}

func testQUIC(addr string, sni string, alpn string) {
	logDiag(fmt.Sprintf("--- TEST C: Protocol Identity (SNI: %s, ALPN: %s) ---", sni, alpn))
	
	tlsConf := &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
		NextProtos:         []string{alpn},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	conn, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		logDiag(fmt.Sprintf("QUIC Handshake FAILED: %v", err))
		return
	}
	defer conn.CloseWithError(0, "")

	logDiag(fmt.Sprintf("QUIC Handshake SUCCESS in %v", time.Since(start)))
}

func main() {
	flag.Parse()
	if *target == "" {
		fmt.Println("Usage: slopn-probe -addr <server-ip>:4242")
		return
	}

	fmt.Printf("SloPN Diagnostic Probe v0.9.5-diag\n")
	fmt.Printf("Target: %s\n\n", *target)

	testUDP(*target)
	fmt.Println()
	testMTU(*target)
	fmt.Println()
	testQUIC(*target, "www.google.com", "h3")
	fmt.Println()
	testQUIC(*target, "www.google.com", "slopn-protocol")
}
