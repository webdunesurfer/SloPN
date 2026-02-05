# Agent Memory: SloPN Project

## Project Overview
- **Name:** SloPN
- **Goal:** Develop a custom Hub-and-Spoke VPN.
- **Protocol:** QUIC (via `quic-go`) utilizing **Unreliable Datagrams (RFC 9221)** to avoid TCP-over-TCP meltdown.
- **Tunneling:** `water` (Go library) + Wintun (for Windows, currently focusing on macOS CLI).
- **UI:** Wails (Go + React/Vue).

## Architectural Decisions (ADRs)
1. **IPAM:** Dynamic IP assignment (DHCP-like) from a server-side pool (e.g., `10.100.0.0/24`). No session persistence.
2. **Authentication:** Single server TLS certificate + custom "Login" JSON message over a reliable QUIC stream immediately after connection.
3. **MTU:** Fixed 1280 bytes on TUN interface to avoid fragmentation complexity while providing headroom for QUIC/UDP/IP headers.
4. **Control Plane:** JSON-based protocol over a reliable bidirectional QUIC stream.
5. **Routing:** Full Tunneling (all traffic through VPN) with server-side NAT/Masquerading.

## Technical Environment
- **OS:** macOS (darwin)
- **Go Path:** `/opt/homebrew/bin/go` (Homebrew installed, but not currently in the environment PATH for this session).
- **Git:** Initialized and connected to `git@github.com:webdunesurfer/SloPN.git`.

## Current Status
- **Phase:** Phase 3 (Hub-and-Spoke Routing)
- **Completed:** Phase 2 (Point-to-Point Tunnel) - TUN interfaces are created and configured (on macOS). IP packets are forwarded over QUIC datagrams.
- **Next Action:** Implement server-side routing logic and dynamic IPAM.

## Key Constraints
- **Unreliable Datagrams:** Must use QUIC Datagrams for VPN traffic.
- **Hub-and-Spoke:** Server must manage multiple client sessions and route between them.
- **Security:** Handshake must be encrypted (TLS 1.3 via QUIC) before token exchange.
