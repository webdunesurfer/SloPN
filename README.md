# SloPN: QUIC-based Layer 3 VPN

SloPN (Slow Private Network) is a modular, high-security VPN built with Go and QUIC. It utilizes RFC 9221 Datagrams to provide a robust tunneling experience that avoids the "TCP-over-TCP" meltdown common in older VPN protocols.

## 📖 Documentation Hub

For detailed information on various aspects of the project, please refer to the following documents:

### 🏛️ Architecture & Design
- **[System Architecture](docs/architecture.md)**: High-level overview of the protocol, data flow, and components.
- **[ADR Index](docs/adr/)**: Architectural Decision Records explaining the *why* behind key technical choices.

### 🛠️ Development & Operations
- **[Build and Release Guide](docs/build-and-release.md)**: Comprehensive instructions for local building, packaging, and CI/CD.
- **[GUI Dashboard](docs/gui-dashboard.md)**: Details on the Wails/Svelte-based user interface.
- **[Frontend Development](docs/frontend-development.md)**: Specifics on the Svelte frontend environment and tooling.
- **[Project Dependencies](docs/dependencies.md)**: Required drivers (WinTUN/TAP) and tools.

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

---

## 🖥️ Server Setup (Docker)

The recommended way to run the SloPN server is via Docker.

### 🚀 One-Click Installation (Linux)
Run this command on your Linux server to automatically install and start the latest SloPN server with a secure random token:
```bash
curl -sSL https://raw.githubusercontent.com/webdunesurfer/SloPN/main/install-server.sh | bash
```

---

## 💻 Component Overview

- **Server (`cmd/server`)**: Linux-native hub (containerized) that manages sessions, IPAM, and NAT.
- **DNS Server (`coredns`)**: Self-hosted private DNS resolver that ensures leak-proof browsing.
- **Helper (`cmd/helper`)**: Unified background service (Engine) for macOS and Windows with authenticated IPC security.
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