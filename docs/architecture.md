# SloPN Architecture

SloPN (Slow Private Network) is a modular, high-security Layer 3 VPN built with Go and QUIC.

## Core Protocols
- **QUIC (RFC 9000):** Primary transport for control and data. It provides the reliability of TCP for signaling and the performance of UDP for tunneling.
- **TLS 1.3:** Built into QUIC, ensuring all traffic is encrypted and authenticated by default.
- **Layer 3 (IP):** The VPN tunnels raw IPv4 packets over QUIC Datagrams (RFC 9221).

## Data Flow
1. **Control Plane:** A reliable QUIC stream is used for the authenticated Login handshake (JSON-based).
2. **Data Plane:** Raw IP packets are intercepted by a virtual TUN interface, wrapped in unreliable QUIC Datagrams, and forwarded to the peer.
3. **Server Routing:** The server acts as a hub, using a Session Manager to route packets between clients or NATing them to the public internet.

## DNS Architecture & Leak Protection
To ensure complete privacy and prevent leaks, SloPN implements a multi-layered DNS infrastructure:
- **Server-Side:** A **CoreDNS** container runs alongside the VPN server. Traffic on port 53 is intercepted via `iptables` and redirected to CoreDNS.
- **macOS Client:** The Helper configures system-wide DNS using `networksetup` while connected.
- **Windows Client:** Aggressive protection is used. The Helper forces the DNS of **all** active network adapters to `10.100.0.1` to prevent Windows from using the ISP's DNS in parallel (Multi-Homed resolution).

## Security & Storage
- **Encryption:** All tunnel traffic is encrypted using TLS 1.3.
- **IPC Security:** GUI-to-Helper communication uses **Local TCP** authenticated via a **Shared Secret** (32-byte hex string).
- **Secure Storage:** 
    - **macOS:** Keychain via `zalando/go-keyring`.
    - **Windows:** Credential Manager via `zalando/go-keyring`.

## OS Specifics
###  macOS
- **Helper:** Background service managed by `launchd`.
- **TUN:** Uses standard `utun` devices.
- **Tray:** Native Cocoa `NSStatusItem` via CGO.

### ⊞ Windows
- **Helper:** Background service managed by **Service Control Manager (SCM)**.
- **TUN:** Uses **TAP-Windows V9** driver with automated discovery and explicit naming (`slopn-tap0`).
- **Tray:** Native Win32 API implementation via `syscall`.

## Component Overview
- **`pkg/protocol`:** QUIC Handshake and control messages.
- **`pkg/ipc`:** Inter-Process Communication.
- **`pkg/tunutil`:** Multi-platform TUN abstraction (`tunutil_windows.go`, `tunutil_darwin.go`).
- **`cmd/helper`:** Unified engine codebase using build tags for platform-specific logic (`platform_windows.go`, etc.).
- **`gui/`:** Svelte + Wails frontend with platform-specific path handling.
