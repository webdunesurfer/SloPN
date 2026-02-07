// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/webdunesurfer/SloPN/pkg/ipc"
)

const GUIVersion = "0.1.9"

// App struct
type App struct {
	ctx context.Context
	mu  sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	fmt.Printf("[v%s] SloPN GUI Starting. PID: %d\n", GUIVersion, os.Getpid())
	a.ctx = ctx
	go a.statusPoller()
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	fmt.Printf("[v%s] SloPN GUI is shutting down...\n", GUIVersion)
}

// GetGUIVersion returns the GUI's version
func (a *App) GetGUIVersion() string {
	return GUIVersion
}

type InitialConfig struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// GetInitialConfig reads the config file created by the installer
func (a *App) GetInitialConfig() InitialConfig {
	path := "/Library/Application Support/SloPN/config.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return InitialConfig{}
	}
	var config InitialConfig
	json.Unmarshal(data, &config)
	return config
}

// ShowAbout displays the application information
func (a *App) ShowAbout() {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "About SloPN",
		Message: fmt.Sprintf("SloPN OS X GUI v%s\n\n© 2026 webdunesurfer\nLicensed under GNU GPLv3", GUIVersion),
	})
}

// callHelper sends a command to the privileged helper with retries
func (a *App) callHelper(req ipc.Request) (*ipc.Response, error) {
	var conn net.Conn
	var err error

	addr := "127.0.0.1:54321"

	for i := 0; i < 3; i++ {
		conn, err = net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		fmt.Printf("[v%s] [DEBUG] IPC connection failure: %v\n", GUIVersion, err)
		return nil, fmt.Errorf("cannot reach helper: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	var resp ipc.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.Status == "error" {
		return nil, fmt.Errorf(resp.Message)
	}

	return &resp, nil
}

// Connect starts the VPN
func (a *App) Connect(server, token string, full bool) string {
	fmt.Printf("[v%s] [GUI] Connect requested for %s\n", GUIVersion, server)
	_, err := a.callHelper(ipc.Request{
		Command:    ipc.CmdConnect,
		ServerAddr: server,
		Token:      token,
		FullTunnel: full,
	})
	if err != nil {
		fmt.Printf("[v%s] [GUI] Connect FAILED: %v\n", GUIVersion, err)
		return err.Error()
	}
	fmt.Printf("[v%s] [GUI] Connect command accepted\n", GUIVersion)
	return "success"
}

// Disconnect stops the VPN
func (a *App) Disconnect() string {
	fmt.Printf("[v%s] [GUI] Disconnect requested\n", GUIVersion)
	_, err := a.callHelper(ipc.Request{Command: ipc.CmdDisconnect})
	if err != nil {
		fmt.Printf("[v%s] [GUI] Disconnect FAILED: %v\n", err)
		return err.Error()
	}
	fmt.Printf("[v%s] [GUI] Disconnect command accepted\n", GUIVersion)
	return "success"
}

// GetStatus returns current VPN status
func (a *App) GetStatus() (*ipc.Status, error) {
	resp, err := a.callHelper(ipc.Request{Command: ipc.CmdGetStatus})
	if err != nil {
		return nil, err
	}
	dataJSON, _ := json.Marshal(resp.Data)
	var status ipc.Status
	json.Unmarshal(dataJSON, &status)
	return &status, nil
}

// GetStats returns real-time stats
func (a *App) GetStats() (*ipc.Stats, error) {
	resp, err := a.callHelper(ipc.Request{Command: ipc.CmdGetStats})
	if err != nil {
		return nil, err
	}
	dataJSON, _ := json.Marshal(resp.Data)
	var stats ipc.Stats
	json.Unmarshal(dataJSON, &stats)
	return &stats, nil
}

// GetLogs returns the last helper logs
func (a *App) GetLogs() (string, error) {
	resp, err := a.callHelper(ipc.Request{Command: ipc.CmdGetLogs})
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}

// statusPoller pushes updates to the frontend every second
func (a *App) statusPoller() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			status, err := a.GetStatus()
			if err != nil {
				runtime.EventsEmit(a.ctx, "helper_status", "missing")
				runtime.EventsEmit(a.ctx, "vpn_status", ipc.Status{State: "disconnected"})
				continue
			}
			
			runtime.EventsEmit(a.ctx, "helper_status", "ok")
			runtime.EventsEmit(a.ctx, "vpn_status", status)
			updateTrayStatus(status.State == "connected")
			
			stats, err := a.GetStats()
			if err == nil {
				runtime.EventsEmit(a.ctx, "vpn_stats", stats)
			}

			logs, err := a.GetLogs()
			if err == nil {
				runtime.EventsEmit(a.ctx, "vpn_logs", logs)
			}
		}
	}
}
