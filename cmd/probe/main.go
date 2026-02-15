package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"
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

	for i := 1; i <= iterations; i++ {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		
		payload := make([]byte, size)
		if iterations > 1 { rand.Read(payload) }
		
		// SloPN Forensic Header: [Marker 0xFF] + [SEQ-000000]
		payload[0] = 0xFF 
		seqStr := fmt.Sprintf("SEQ-%06d", i)
		copy(payload[1:], []byte(seqStr))

		start := time.Now()
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(2500 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		
		if err == nil && n == size && string(buf[1:12]) == seqStr {
			success++
			totalTime += time.Since(start)
		} else if err != nil {
			logTest(name, fmt.Sprintf("  Packet %d: TIMEOUT", i))
		}
		conn.Close()
		time.Sleep(100 * time.Millisecond)
	}

	loss := float64(iterations-success) / float64(iterations) * 100
	avg := time.Duration(0)
	if success > 0 { avg = totalTime / time.Duration(success) }
	logTest(name, fmt.Sprintf("RESULT: %d/%d received | Loss: %.1f%% | Avg RTT: %v", success, iterations, loss, avg))
}

func testFlow(addr string) {
	logTest("FLOW-TEST", "Starting 60-second forensic flow test (1 packet/sec)...")
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	conn, _ := net.DialUDP("udp", nil, udpAddr)
	defer conn.Close()

	for i := 1; i <= 60; i++ {
		payload := make([]byte, 64)
		payload[0] = 0xFF
		seqStr := fmt.Sprintf("FLW-%06d", i)
		copy(payload[1:12], []byte(seqStr))
		
		conn.Write(payload)
		
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		_, _, err := conn.ReadFromUDP(buf)
		
		if err != nil {
			logTest("FLOW-TEST", fmt.Sprintf("  Packet %d (%s): DROPPED", i, seqStr))
		} else if i % 10 == 0 {
			logTest("FLOW-TEST", fmt.Sprintf("  Flow healthy at %ds (Packet %d)...", i, i))
		}
		time.Sleep(1 * time.Second)
	}
	logTest("FLOW-TEST", "Forensic flow test complete.")
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
		fmt.Printf("Saving forensic results to: %s\n", fileName)
	}

	printf("SloPN Diagnostic Probe v0.9.5-diag-v11 (Forensic Edition)\n")
	printf("Target: %s\n", *target)
	printf("====================================================\n")

	runUDPTest("BASELINE", *target, 64, "Forensic Ping", 5)
	printf("\n")

	testFlow(*target)
	printf("\n")

	fmt.Println("Diagnostic Complete. Please provide the console output AND the server logs.")
}
