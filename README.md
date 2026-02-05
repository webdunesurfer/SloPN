# SloPN: Custom QUIC-based VPN

SloPN is a high-performance "Hub-and-Spoke" VPN built with Go and QUIC (RFC 9221 Datagrams).

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

## 💻 Client Setup
The client connects to the server and creates a local TUN interface.

### Build
```bash
go build -o client ./cmd/client
```

### Configuration
Create a `config.json` in the client directory:
```json
{
  "server_addr": "your-server-ip:4242",
  "token": "your-secret-token"
}
```

### Run
```bash
sudo ./client [flags]
```

### Options
- `-v`: Enable verbose logging.
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
*Maintained by webdunesurfer*