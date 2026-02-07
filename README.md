# SloPN: QUIC-based Layer 3 VPN

SloPN (Slow Private Network) is a modular, high-security VPN built with Go and QUIC. It utilizes RFC 9221 Datagrams to provide a robust tunneling experience that avoids the "TCP-over-TCP" meltdown common in older VPN protocols.

## 📖 Documentation Hub

For detailed information on various aspects of the project, please refer to the following documents:

### 🏛️ Architecture & Design
- **[System Architecture](docs/Architecture.md)**: High-level overview of the protocol, data flow, and components.
- **[ADR Index](docs/adr/)**: Architectural Decision Records explaining the *why* behind key technical choices.

### 🛠️ Development & Operations
- **[Build and Release Guide](docs/build-and-release.md)**: Comprehensive instructions for local building, packaging, and CI/CD.
- **[GUI Dashboard](docs/gui-dashboard.md)**: Details on the Wails/Svelte-based user interface.
- **[Frontend Development](docs/frontend-development.md)**: Specifics on the Svelte frontend environment and tooling.
- **[Refactoring Plan](docs/RefactoringPlan.md)**: Current roadmap for code improvements and technical debt.

### 📈 Project Roadmap
- **[Development Plan](docs/plan.md)**: Overall strategy and high-level milestones.
- **[Implementation Phases](docs/phases/)**: Step-by-step breakdown of the project's evolution from transport to GUI and containerization.

---

## 🚀 Quick Start (macOS)

### 1. Installation via Installer
The easiest way to get started on macOS is using the pre-built installer:
1. Download `SloPN-Installer.pkg` from the [Latest Release](https://github.com/webdunesurfer/SloPN/releases).
2. Run the installer (requires administrator privileges).
3. Open **SloPN** from your Applications folder.

### 2. Manual Development Setup
```bash
git clone https://github.com/webdunesurfer/SloPN.git
cd SloPN
go mod tidy
```
Refer to the **[Build and Release Guide](docs/build-and-release.md)** for compilation instructions.

---

## 💻 Component Overview

- **Server (`cmd/server`)**: Linux-native hub that manages sessions, IPAM, and NAT.
- **Helper (`cmd/helper`)**: Privileged background service (Engine) that manages system networking.
- **GUI (`gui/`)**: User-space dashboard for controlling the connection.
- **CLI Control (`cmd/slopnctl`)**: Lightweight tool for interacting with the helper via terminal.

## 🧪 Testing Connectivity
Once connected, verify the tunnel using standard tools:
```bash
# Ping the server virtual IP from client
ping 10.100.0.1

# Ping a client virtual IP from server
ping 10.100.0.2
```

---
**Author:** webdunesurfer <vkh@gmx.at>  
**License:** [GNU GPLv3](LICENSE)