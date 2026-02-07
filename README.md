# SloPN: Custom QUIC-based VPN

SloPN is a ~~high~~ slow-performance "Hub-and-Spoke" VPN built with Go and QUIC (RFC 9221 Datagrams).

## 🚀 Getting Started

### Prerequisites
- **Go 1.21+**
- **Root/Admin Privileges:** Required to create virtual TUN interfaces.
- **Linux:** Requires `ip` command and `sysctl` capabilities.
- **macOS:** Requires standard `ifconfig` and `route` commands.

### Installation
```bash
git clone https://github.com/webdunesurfer/SloPN.git
cd SloPN
go mod tidy
```

## 🖥️ Server Setup (Linux/macOS)
The server handles IP allocation (IPAM) and routes traffic between clients.

### Build
```bash
go build -o server ./cmd/server
```

### Run
```bash
sudo ./server [flags]
```

### Options
- `-v`: Enable verbose logging (shows every packet summary).
- `-ip string`: Server Virtual IP (default "10.100.0.1").
- `-subnet string`: VPN Subnet CIDR (default "10.100.0.0/24").
- `-port int`: UDP Port to listen on (default 4242).

## 💻 Client & GUI Setup
The client architecture has evolved into a **Privileged Helper** and a **Svelte-based GUI**.

### Build Helper (Engine)
```bash
go build -o slopn-helper ./cmd/helper
sudo ./slopn-helper
```

### Build GUI (macOS/Windows/Linux)
Requires [Wails](https://wails.io/):
```bash
cd gui
wails build
```

### Versioning
All components (Server, Helper, GUI) now support unified versioning (currently `v0.1.1`) displayed in the GUI dashboard.

### Configuration
The GUI allows real-time configuration of:
- **Server Address**: IP and Port of the SloPN server.
- **Auth Token**: Secure token for connection.
- **Full Tunnel**: Toggle between routing all traffic or just VPN traffic.

- `server_addr`: Public IP and Port of the SloPN server.
- `token`: Authentication token.
- `verbose`: If true, logs packet flow to console.
- `host_route_only`: If true, only routes the Server VIP through the tunnel (useful for multi-client testing on one machine).
- `no_route`: If true, does not modify the system routing table at all.

### Run
```bash
sudo ./client [flags]
```

### Options
- `-v`: Force verbose logging (overrides config).
- `-config string`: Path to config file (default "config.json").

## 🧪 Testing Connectivity
Once connected, you can verify the tunnel using standard tools:
```bash
# Ping the server from client
ping 10.100.0.1

# Ping a client from server
ping 10.100.0.2
```

---
Author: webdunesurfer <vkh@gmx.at>

This project is licensed under the GNU GPLv3 - see the [LICENSE](LICENSE) file for details.

