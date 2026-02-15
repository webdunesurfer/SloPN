package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	target = flag.String("addr", "", "Server address (e.g. 1.2.3.4:4242)")
)

func logDiag(tag, msg string) {
	fmt.Printf("[%s] [%-10s] %s\n", time.Now().Format("15:04:05"), tag, msg)
}

func testBaseline(addr string) {
	logDiag("BASE", "--- TEST A: UDP Connectivity & Jitter ---")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	
	var latencies []time.Duration
	for i := 1; i <= 5; i++ {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		start := time.Now()
		conn.Write([]byte(fmt.Sprintf("PING-%d", i)))
		
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			logDiag("BASE", fmt.Sprintf("Probe %d: TIMEOUT", i))
		} else {
			d := time.Since(start)
			latencies = append(latencies, d)
			logDiag("BASE", fmt.Sprintf("Probe %d: OK (%v)", i, d))
		}
		conn.Close()
		time.Sleep(100 * time.Millisecond)
	}
}

func testMTU(addr string) {
	logDiag("MTU", "--- TEST B: MTU Sweep ---")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	sizes := []int{500, 1000, 1200, 1300, 1400, 1450}

	for _, sz := range sizes {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		payload := make([]byte, sz)
		copy(payload, []byte(fmt.Sprintf("MTU-%d", sz)))
		
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		_, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			logDiag("MTU", fmt.Sprintf("Size %4d bytes: [FAILED]", sz))
		} else {
			logDiag("MTU", fmt.Sprintf("Size %4d bytes: [SUCCESS]", sz))
		}
		conn.Close()
	}
}

func testEntropy(addr string) {
	logDiag("ENTROPY", "--- TEST C: High vs Low Entropy ---")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)

	// Low Entropy (Zeros)
	conn1, _ := net.DialUDP("udp", nil, udpAddr)
	lowPayload := make([]byte, 800)
	start1 := time.Now()
	conn1.Write(lowPayload)
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err1 := conn1.ReadFromUDP(make([]byte, 1024))
	logDiag("ENTROPY", fmt.Sprintf("Low Entropy (800b Zeros):  %v", err1 == nil))
	conn1.Close()

	// High Entropy (Random)
	conn2, _ := net.DialUDP("udp", nil, udpAddr)
	highPayload := make([]byte, 800)
	rand.Read(highPayload)
	start2 := time.Now()
	conn2.Write(highPayload)
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err2 := conn2.ReadFromUDP(make([]byte, 1024))
	logDiag("ENTROPY", fmt.Sprintf("High Entropy (800b Random): %v", err2 == nil))
	conn2.Close()
	
	_ = start1
	_ = start2
}

func testQUIC(addr, alpn string) {
	logDiag("QUIC", fmt.Sprintf("--- TEST D: Handshake (ALPN: %s) ---", alpn))
	
	tlsConf := &tls.Config{
		ServerName:         "www.google.com",
		InsecureSkipVerify: true,
		NextProtos:         []string{alpn},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	conn, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		logDiag("QUIC", fmt.Sprintf("Result: FAILED (%v)", err))
		return
	}
	defer conn.CloseWithError(0, "")

	logDiag("QUIC", fmt.Sprintf("Result: SUCCESS (Time: %v)", time.Since(start)))
}

func main() {
	flag.Parse()
	if *target == "" {
		fmt.Println("Usage: slopn-probe -addr <server-ip>:4242")
		return
	}

	fmt.Printf("SloPN Diagnostic Probe v0.9.5-diag-v3\n")
	fmt.Printf("Target: %s\n", *target)
	fmt.Println("====================================================")

	testBaseline(*target)
	fmt.Println()
	testMTU(*target)
	fmt.Println()
	testEntropy(*target)
	fmt.Println()
	testQUIC(*target, "h3")
	testQUIC(*target, "slopn-protocol")
	testQUIC(*target, "http/1.1")
}
