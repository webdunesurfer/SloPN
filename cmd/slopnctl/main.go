// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/webdunesurfer/SloPN/pkg/ipc"
)

func main() {
	cmd := flag.String("cmd", "status", "Command: connect, disconnect, status, stats")
	server := flag.String("server", "", "Server address (for connect)")
	token := flag.String("token", "", "Auth token (for connect)")
	full := flag.Bool("full", false, "Enable full tunnel (for connect)")
	flag.Parse()

	var command ipc.Command
	switch *cmd {
	case "connect":
		command = ipc.CmdConnect
	case "disconnect":
		command = ipc.CmdDisconnect
	case "status":
		command = ipc.CmdGetStatus
	case "stats":
		command = ipc.CmdGetStats
	default:
		log.Fatalf("Unknown command: %s", *cmd)
	}

	conn, err := net.Dial("unix", "/tmp/slopn.sock")
	if err != nil {
		log.Fatalf("Failed to connect to helper: %v. Is it running?", err)
	}
	defer conn.Close()

	req := ipc.Request{
		Command:    command,
		ServerAddr: *server,
		Token:      *token,
		FullTunnel: *full,
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		log.Fatal(err)
	}

	var resp ipc.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		log.Fatal(err)
	}

	if resp.Status == "error" {
		fmt.Printf("Error: %s\n", resp.Message)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", resp.Status)
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}
	if resp.Data != nil {
		dataJSON, _ := json.MarshalIndent(resp.Data, "", "  ")
		fmt.Printf("Data: %s\n", string(dataJSON))
	}
}