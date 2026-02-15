package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
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

func runUDPTest(name string, conn *net.UDPConn, size int, label string, iterations int) {
	success := 0
	var totalTime time.Duration

	logTest(name, fmt.Sprintf("Starting %s (%d bytes, %d probes)...", label, size, iterations))

	for i := 1; i <= iterations; i++ {
		payload := make([]byte, size)
		if size > 32 { rand.Read(payload[32:]) }
		
		payload[0] = 0xFF 
		seqStr := fmt.Sprintf("SEQ-%06d", i)
		copy(payload[1:11], []byte(seqStr))
		
		checksum := crc32.ChecksumIEEE(payload[:size-4])
		binary.BigEndian.PutUint32(payload[size-4:], checksum)

		start := time.Now()
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := conn.Read(buf)
		
		if err == nil {
			receivedCRC := binary.BigEndian.Uint32(buf[n-4:n])
			computedCRC := crc32.ChecksumIEEE(buf[:n-4])
			if n == size && string(buf[1:11]) == seqStr && receivedCRC == computedCRC {
				success++
				totalTime += time.Since(start)
			} else {
				logTest(name, fmt.Sprintf("  %s: ERROR (Size: %d, Int: %v)", seqStr, n, receivedCRC == computedCRC))
			}
		} else {
			logTest(name, fmt.Sprintf("  %s: TIMEOUT", seqStr))
		}
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
		logTest("QUIC-PROBE", fmt.Sprintf("Handshake info: %v", err))
		return
	}
	defer conn.CloseWithError(0, "")
	logTest("QUIC-PROBE", fmt.Sprintf("Handshake SUCCESS in %v", time.Since(start)))
}

func testFlow(name string, conn *net.UDPConn, size int, highEntropy bool) {
	label := "Low Entropy"
	if highEntropy { label = "High Entropy" }
	logTest(name, fmt.Sprintf("Starting 60s %s test (%d bytes, 1 pps)...", label, size))

	for i := 1; i <= 60; i++ {
		payload := make([]byte, size)
		if highEntropy { rand.Read(payload) }
		payload[0] = 0xFF
		seqStr := fmt.Sprintf("FLW-%06d", i)
		copy(payload[1:11], []byte(seqStr))
		
		checksum := crc32.ChecksumIEEE(payload[:size-4])
		binary.BigEndian.PutUint32(payload[size-4:], checksum)
		
		conn.Write(payload)
		
		buf := make([]byte, 2048)
		conn.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))
		n, err := conn.Read(buf)
		
		if err == nil {
			receivedCRC := binary.BigEndian.Uint32(buf[n-4:n])
			computedCRC := crc32.ChecksumIEEE(buf[:n-4])
			if string(buf[1:11]) == seqStr && receivedCRC == computedCRC {
				if i % 10 == 0 {
					logTest(name, fmt.Sprintf("  %s flow healthy at %ds...", label, i))
				}
			} else {
				logTest(name, fmt.Sprintf("  Packet %d: INTEGRITY FAILED", i))
			}
		} else {
			logTest(name, fmt.Sprintf("  Packet %d (%s): TIMEOUT", i, seqStr))
		}
		time.Sleep(1 * time.Second)
	}
	logTest(name, fmt.Sprintf("%s test complete.", label))
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
	if err == nil {
		defer f.Close()
		out = io.MultiWriter(os.Stdout, f)
		fmt.Printf("Saving results to: %s\n", fileName)
	} else {
		out = os.Stdout
	}

	printf("SloPN Diagnostic Probe v0.9.5-diag-v18 (FULL Forensic Suite)\n")
	printf("Target: %s\n", *target)
	printf("====================================================\n")

	udpAddr, _ := net.ResolveUDPAddr("udp", *target)
	conn, _ := net.DialUDP("udp", nil, udpAddr)
	defer conn.Close()

	// 1. Baseline
	runUDPTest("BASELINE", conn, 64, "Forensic Ping", 5)
	printf("\n")

	// 2. MTU Sweep
	sizes := []int{500, 1200, 1400}
	for _, s := range sizes {
		runUDPTest("MTU-SWEEP", conn, s, fmt.Sprintf("%d bytes", s), 1)
	}
	printf("\n")

	// 3. Handshake
	testQUIC(*target, "h3")
	printf("\n")

	// 4. Flow Low Entropy
	testFlow("FLOW-LOW", conn, 64, false)
	printf("\n")

	// 5. Flow High Entropy
	testFlow("FLOW-HIGH", conn, 1000, true)
}
