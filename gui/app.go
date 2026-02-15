// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
	"github.com/webdunesurfer/SloPN/pkg/ipc"
	"net/http"
)

const (
	GUIVersion = "0.9.5-diag-v7"
	Service    = "com.webdunesurfer.slopn"
	Account    = "auth_token"
)

// IPInfo represents public IP and geolocation data
type IPInfo struct {
	Query       string `json:"query"`
	City        string `json:"city"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	ISP         string `json:"isp"`
}

// App struct
type App struct {
	ctx       context.Context
	mu        sync.Mutex
	ipcSecret string
}

func (a *App) loadIPCSecret() {
	data, err := os.ReadFile(SecretPath)
	if err != nil {
		fmt.Printf("[v%s] [WARNING] Could not read IPC secret from %s: %v\n", GUIVersion, SecretPath, err)
		return
	}
	a.ipcSecret = strings.TrimSpace(string(data))
	fmt.Printf("[v%s] [GUI] IPC Secret loaded.\n", GUIVersion)
}

// UserSettings for non-sensitive data
type UserSettings struct {
	Server     string `json:"server"`
	FullTunnel bool   `json:"full_tunnel"`
	Obfuscate  bool   `json:"obfuscate"`
	SNI        string `json:"sni"`
}

// SaveConfig persists settings to disk and token to Keyring
func (a *App) SaveConfig(server, token, sni string, full, obfs bool) {
	server = strings.TrimSpace(server)
	sni = strings.TrimSpace(sni)
	// 1. Save sensitive token to system Keyring
	if token != "" {
		err := keyring.Set(Service, Account, token)
		if err != nil {
			fmt.Printf("[v%s] [ERROR] Keyring save failed: %v\n", GUIVersion, err)
		} else {
			fmt.Printf("[v%s] [GUI] Token updated in Keyring\n", GUIVersion)
		}
	} else {
		// If token is empty, try to remove it from keyring to clear it
		keyring.Delete(Service, Account)
		fmt.Printf("[v%s] [GUI] Token cleared from Keyring\n", GUIVersion)
	}

	// 2. Save non-sensitive settings to User Home
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	
	settings := UserSettings{Server: server, FullTunnel: full, Obfuscate: obfs, SNI: sni}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(configDir, "settings.json"), data, 0644)
	fmt.Printf("[v%s] [GUI] Config (Server: %s, Full: %v, Obfs: %v, SNI: %s) saved to Library\n", GUIVersion, server, full, obfs, sni)
}

// GetSavedConfig retrieves settings and secure token
func (a *App) GetSavedConfig() map[string]interface{} {
	res := make(map[string]interface{})

	// 1. Load settings from JSON
	configDir := getConfigDir()
	settingsPath := filepath.Join(configDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var settings UserSettings
		if err := json.Unmarshal(data, &settings); err == nil {
			res["server"] = settings.Server
			res["full_tunnel"] = settings.FullTunnel
			res["obfuscate"] = settings.Obfuscate
			res["sni"] = settings.SNI
		}
	}

	// 2. Load token from Keyring
	if token, err := keyring.Get(Service, Account); err == nil {
		res["token"] = token
	}

	return res
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}
	a.loadIPCSecret()
	return a
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

// CheckNewInstall returns true if the .new_install marker exists, then deletes it
func (a *App) CheckNewInstall() bool {
	if _, err := os.Stat(NewInstallMarkerPath); err == nil {
		os.Remove(NewInstallMarkerPath)
		fmt.Printf("[v%s] [GUI] New installation marker detected. Prioritizing installer config.\n", GUIVersion)
		return true
	}
	return false
}

type InitialConfig struct {
	Server    string `json:"server"`
	Token     string `json:"token"`
	Obfuscate interface{} `json:"obfuscate"` // Handle both bool and string
	SNI       string `json:"sni"`
}

// GetInitialConfig reads the config file created by the installer
func (a *App) GetInitialConfig() InitialConfig {
	data, err := os.ReadFile(InstallConfigPath)
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

	req.IPCSecret = a.ipcSecret

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
func (a *App) Connect(server, token, sni string, full, obfs bool) string {
	server = strings.TrimSpace(server)
	sni = strings.TrimSpace(sni)
	fmt.Printf("[v%s] [GUI] Connect requested for %s (SNI: %s, Obfs: %v)\n", GUIVersion, server, sni, obfs)
	_, err := a.callHelper(ipc.Request{
		Command:    ipc.CmdConnect,
		ServerAddr: server,
		Token:      token,
		SNI:        sni,
		FullTunnel: full,
		Obfuscate:  obfs,
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
	
	// Use json.Marshal/Unmarshal to convert map[string]interface{} to *ipc.Status
	// but ensuring we return the pointer so Wails maps it to JS object correctly.
	dataJSON, _ := json.Marshal(resp.Data)
	var status ipc.Status
	if err := json.Unmarshal(dataJSON, &status); err != nil {
		return nil, err
	}
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
	if err := json.Unmarshal(dataJSON, &stats); err != nil {
		return nil, err
	}
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

// GetPublicIPInfo fetches the current public IP and location
func (a *App) GetPublicIPInfo() (*IPInfo, error) {
	// Create a transport that disables connection reuse (Keep-Alives)
	// This ensures that when the VPN connects, we don't reuse a socket
	// that was established over the physical interface.
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	
	// Add timestamp to bypass server-side caches
	url := fmt.Sprintf("http://ip-api.com/json?t=%d", time.Now().Unix())
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
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
