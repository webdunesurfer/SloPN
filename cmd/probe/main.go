package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	target = flag.String("addr", "", "Server address (e.g. 1.2.3.4:4242)")
	out    io.Writer
)

func printf(format string, a ...interface{}) {
	fmt.Fprintf(out, format, a...)
}

func logTest(name, msg string) {
	printf("[%s] [%-12s] %s\n", time.Now().Format("15:04:05"), name, msg)
}

func runUDPTest(name string, addr string, size int, label string, iterations int) {
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	success := 0
	var totalTime time.Duration

	logTest(name, fmt.Sprintf("Starting %s (%d bytes, %d probes)...", label, size, iterations))

	for i := 0; i < iterations; i++ {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		
		payload := make([]byte, size)
		if iterations > 1 { rand.Read(payload) }
		payload[0] = 0xFF // Explicit Diag Echo Marker for SloPN Diag-v8

		start := time.Now()
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		
		if err == nil && n == size && buf[0] == 0xFF {
			success++
			totalTime += time.Since(start)
		}
		conn.Close()
		time.Sleep(50 * time.Millisecond)
	}

	loss := float64(iterations-success) / float64(iterations) * 100
	avg := time.Duration(0)
	if success > 0 { avg = totalTime / time.Duration(success) }
	logTest(name, fmt.Sprintf("RESULT: %d/%d received | Loss: %.1f%% | Avg RTT: %v", success, iterations, loss, avg))
}

func testQUIC(addr, alpn string) {
	logTest("QUIC-PROBE", fmt.Sprintf("Testing Handshake Mirroring (ALPN: %s)...", alpn))
	tlsConf := &tls.Config{
		ServerName: "www.google.com",
		InsecureSkipVerify: true,
		NextProtos: []string{alpn},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	conn, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		// In diagnostic mode, we just want to see if we get a response
		logTest("QUIC-PROBE", fmt.Sprintf("Handshake info: %v", err))
		return
	}
	defer conn.CloseWithError(0, "")
	logTest("QUIC-PROBE", fmt.Sprintf("Handshake SUCCESS in %v (Identity Mirroring OK)", time.Since(start)))
}

func testFlow(addr string) {
	logTest("FLOW-TEST", "Starting 60-second continuous flow test (1 packet/sec)...")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	conn, _ := net.DialUDP("udp", nil, udpAddr)
	defer conn.Close()

	for i := 1; i <= 60; i++ {
		payload := make([]byte, 32)
		payload[0] = 0xFF
		copy(payload[1:], []byte(fmt.Sprintf("FLOW-%02d", i)))
		
		conn.Write(payload)
		
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, _, err := conn.ReadFromUDP(buf)
		
		if err != nil {
			logTest("FLOW-TEST", fmt.Sprintf("Packet %d: DROPPED at %ds", i, i))
		} else if i % 10 == 0 {
			logTest("FLOW-TEST", fmt.Sprintf("Flow healthy at %ds...", i))
		}
		time.Sleep(1 * time.Second)
	}
	logTest("FLOW-TEST", "Continuous flow test complete.")
}

func main() {
	flag.Parse()
	if *target == "" {
		fmt.Println("Usage: slopn-probe -addr <server-ip>:4242")
		return
	}

	timestamp := time.Now().Format("20060102-150405")
	fileName := fmt.Sprintf("probing-%s.txt", timestamp)
	f, err := os.Create(fileName)
	if err != nil {
		out = os.Stdout
	} else {
		defer f.Close()
		out = io.MultiWriter(os.Stdout, f)
		fmt.Printf("Saving results to: %s\n", fileName)
	}

	printf("SloPN Diagnostic Probe v0.9.5-diag-v8\n")
	printf("Target: %s\n", *target)
	printf("====================================================\n")

	runUDPTest("BASELINE", *target, 32, "Ping", 5)
	printf("\n")

	sizes := []int{500, 1200, 1400}
	for _, s := range sizes {
		runUDPTest("MTU-SWEEP", *target, s, fmt.Sprintf("%d bytes", s), 1)
	}
	printf("\n")

	testQUIC(*target, "h3")
	printf("\n")

	testFlow(*target)
}
