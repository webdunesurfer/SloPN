// Author: webdunesurfer <vkh@gmx.at>
// Licensed under the GNU General Public License v3.0

package ipc

type Command string

const (
	CmdConnect    Command = "connect"
	CmdDisconnect Command = "disconnect"
	CmdGetStatus  Command = "get_status"
	CmdGetStats   Command = "get_stats"
	CmdGetLogs    Command = "get_logs"
)

type Request struct {
	Command    Command `json:"command"`
	IPCSecret  string  `json:"ipc_secret,omitempty"`
	ServerAddr string  `json:"server_addr,omitempty"`
	Token      string  `json:"token,omitempty"`
	FullTunnel bool    `json:"full_tunnel,omitempty"`
}

type Response struct {
	Status  string      `json:"status"` // "success", "error", "connected", "disconnected"
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Stats struct {
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
	Uptime    int64  `json:"uptime_seconds"`
}

type Status struct {
	State         string `json:"state"` // "connected", "disconnected", "connecting"
	AssignedVIP   string `json:"assigned_vip,omitempty"`
	ServerVIP     string `json:"server_vip,omitempty"`
	ServerAddr    string `json:"server_addr,omitempty"`
	HelperVersion string `json:"helper_version,omitempty"`
	ServerVersion string `json:"server_version,omitempty"`
}
