package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var (
	target = flag.String("addr", "", "Server address (e.g. 1.2.3.4:4242)")
)

func logTest(name, msg string) {
	fmt.Printf("[%s] [%-12s] %s\n", time.Now().Format("15:04:05"), name, msg)
}

func runUDPTest(name string, addr string, size int, label string, iterations int) {
	udpAddr, _ := net.ResolveUDPAddr("udp", addr)
	success := 0
	var totalTime time.Duration

	logTest(name, fmt.Sprintf("Starting %s (%d bytes, %d probes)...", label, size, iterations))

	for i := 0; i < iterations; i++ {
		conn, _ := net.DialUDP("udp", nil, udpAddr)
		
		payload := make([]byte, size)
		if size > 10 {
			copy(payload, []byte("PROBE"))
		}
		if iterations > 1 {
			// Add some randomness to each iteration to keep entropy high
			rand.Read(payload[size/2:])
		}

		start := time.Now()
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		
		if err == nil && n == size {
			success++
			totalTime += time.Since(start)
		}
		conn.Close()
		time.Sleep(50 * time.Millisecond)
	}

	loss := float64(iterations-success) / float64(iterations) * 100
	avg := time.Duration(0)
	if success > 0 {
		avg = totalTime / time.Duration(success)
	}

	status := "PASS"
	if loss > 20 { status = "WARNING" }
	if success == 0 { status = "FAIL" }

	logTest(name, fmt.Sprintf("RESULT: %s | Loss: %.1f%% | Avg RTT: %v", status, loss, avg))
}

func main() {
	flag.Parse()
	if *target == "" {
		fmt.Println("Usage: slopn-probe -addr <server-ip>:4242")
		return
	}

	fmt.Printf("SloPN Diagnostic Probe v0.9.5-diag-v4\n")
	fmt.Printf("Target: %s\n", *target)
	fmt.Println("====================================================")

	// Test A: Low-level Baseline
	runUDPTest("BASELINE", *target, 32, "Tiny Ping", 5)
	fmt.Println()

	// Test B: MTU Sweep
	sizes := []int{500, 1000, 1200, 1300, 1400, 1450}
	for _, s := range sizes {
		runUDPTest("MTU-SWEEP", *target, s, fmt.Sprintf("%d bytes", s), 1)
	}
	fmt.Println()

	// Test C: Entropy Impact (Zero vs Random)
	logTest("ENTROPY", "Testing Zero-filled packets...")
	runUDPTest("ENTROPY-ZERO", *target, 1000, "1000b Zeros", 3)
	
	logTest("ENTROPY", "Testing Random-filled packets...")
	runUDPTest("ENTROPY-RAND", *target, 1000, "1000b Random", 3)
	fmt.Println()

	fmt.Println("Diagnostic Complete. Please provide the output above and the server logs.")
}
