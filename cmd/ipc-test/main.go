package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/webdunesurfer/SloPN/pkg/ipc"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ipc-test <server> <token>")
		return
	}
	server := os.Args[1]
	token := os.Args[2]

	conn, err := net.Dial("tcp", "127.0.0.1:54321")
	if err != nil {
		fmt.Printf("Failed to connect to helper: %v\n", err)
		return
	}
	defer conn.Close()

	req := ipc.Request{
		Command:    ipc.CmdConnect,
		ServerAddr: server,
		Token:      token,
		FullTunnel: false, // Split tunnel for safety first
		IPCSecret:  "test-secret-123",
	}

	fmt.Printf("Sending connect request to %s...\n", server)
	json.NewEncoder(conn).Encode(req)

	var resp ipc.Response
	json.NewDecoder(conn).Decode(&resp)
	fmt.Printf("Response: %+v\n", resp)

	if resp.Status == "success" {
		fmt.Println("Waiting 10 seconds for stats...")
		time.Sleep(10 * time.Second)
		
		// Ask for stats
		conn2, _ := net.Dial("tcp", "127.0.0.1:54321")
		reqStats := ipc.Request{Command: ipc.CmdGetStats, IPCSecret: "test-secret-123"}
		json.NewEncoder(conn2).Encode(reqStats)
		var respStats ipc.Response
		json.NewDecoder(conn2).Decode(&respStats)
		fmt.Printf("Stats: %+v\n", respStats.Data)
		conn2.Close()
	}
}