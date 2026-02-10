# SloPN: QUIC-based Layer 3 VPN

SloPN (Slow Private Network) is a modular, high-security VPN built with Go and QUIC. It utilizes RFC 9221 Datagrams to provide a robust tunneling experience that avoids the "TCP-over-TCP" meltdown common in older VPN protocols.

## 📖 Documentation Hub

For detailed information on various aspects of the project, please refer to the following documents:

### 🏛️ Architecture & Design
- **[System Architecture](docs/architecture.md)**: High-level overview of the protocol, data flow, and components.
- **[ADR Index](docs/adr/)**: Architectural Decision Records explaining the *why* behind key technical choices.

### 🛠️ Development & Operations
- **[Build and Release Guide](docs/build-and-release.md)**: Comprehensive instructions for local building, packaging, and CI/CD.
- **[Project Dependencies](docs/dependencies.md)**: Required drivers (WinTUN/TAP) and tools.
- **[GUI Dashboard](docs/gui-dashboard.md)**: Details on the Wails/Svelte-based user interface.
- **[Frontend Development](docs/frontend-development.md)**: Specifics on the Svelte frontend environment and tooling.
- **[Refactoring Plan](docs/refactoring-plan.md)**: Current roadmap for code improvements and technical debt.

### 📈 Project Roadmap
- **[Development Plan](docs/plan.md)**: Overall strategy and high-level milestones.
- **[Implementation Phases](docs/phases/)**: Step-by-step breakdown of the project's evolution from transport to private DNS infrastructure and security hardening.

---

## 🚀 Quick Start

### ⊞ Windows
1. Download `SloPN-Setup.exe` from the [Latest Release](https://github.com/webdunesurfer/SloPN/releases).
2. Run the installer (requires Administrator privileges for driver and service setup).
3. Enter your Server Address and Token during the installation wizard.
4. Launch **SloPN** from your desktop or Start menu.

###  macOS
1. Download `SloPN-Installer.pkg` from the [Latest Release](https://github.com/webdunesurfer/SloPN/releases).
2. Run the installer (requires administrator privileges).
3. Open **SloPN** from your Applications folder.

### Manual Development Setup
```bash
git clone https://github.com/webdunesurfer/SloPN.git
cd SloPN
go mod tidy
```
Refer to the **[Build and Release Guide](docs/build-and-release.md)** for compilation instructions.

## 🖥️ Server Setup (Docker)

The recommended way to run the SloPN server is via Docker.

### 🚀 One-Click Installation (Linux)
Run this command on your Linux server to automatically install and start the latest SloPN server with a secure random token:
```bash
curl -sSL https://raw.githubusercontent.com/webdunesurfer/SloPN/main/install-server.sh | bash
```

### 1. Using Docker Compose
```bash
# Clone the repository
git clone https://github.com/webdunesurfer/SloPN.git
cd SloPN

# Start both VPN and DNS services
docker compose up -d
```

### 2. Manual Docker Run
If not using compose, you must start both containers. Note that the DNS container should bind to your Docker bridge IP (usually `172.17.0.1`):
```bash
# Start VPN Server
docker run -d \
  --name slopn-server \
  --cap-add=NET_ADMIN \
  --device=/dev/net/tun:/dev/net/tun \
  -p 4242:4242/udp \
  -e SLOPN_TOKEN=your-secret-token \
  slopn-server -nat

# Start DNS Server (Binding to Docker bridge IP)
docker run -d \
  --name slopn-dns \
  -p 172.17.0.1:53:53/udp \
  -p 172.17.0.1:53:53/tcp \
  -v $(pwd)/coredns.conf:/etc/coredns/Corefile \
  coredns/coredns:latest -conf /etc/coredns/Corefile
```

---

## 💻 Component Overview

- **Server (`cmd/server`)**: Linux-native hub (containerized) that manages sessions, IPAM, and NAT.
- **DNS Server (`coredns`)**: Self-hosted private DNS resolver that ensures leak-proof browsing.
- **Helper (`cmd/helper`)**: Privileged background service (Engine) for macOS and Windows with authenticated IPC security.
- **GUI (`gui/`)**: User-space dashboard with platform-native secure storage (macOS Keychain / Windows Credential Manager).

## 🧪 Testing Connectivity
Once connected, verify the tunnel using standard tools:
```bash
# Ping the server virtual IP from client
ping 10.100.0.1

# Verify public IP matches VPN server
curl https://api.ipify.org
```

---
**Author:** webdunesurfer <vkh@gmx.at>  
**License:** [GNU GPLv3](LICENSE)
