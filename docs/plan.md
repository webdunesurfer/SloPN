# Implementation Plan: Custom QUIC-based VPN (Hub-and-Spoke)

## Overview
**Project Name:** SloPN
**Architecture:** Hub-and-Spoke
**Stack:** Go, `quic-go` (RFC 9221 Datagrams), `water` (TUN), Wails.

This plan is divided into distinct phases. Each phase builds upon the previous one and results in a deployable, testable artifact.

## Phase 1: The Transport Layer (QUIC Datagrams)
**Goal:** Establish a secure QUIC connection and exchange unreliable datagrams between a CLI client and server. No TUN interfaces yet.

1.  **Setup Project Structure:** Initialize Go module.
2.  **Certificates:** Generate self-signed CA and certificates for TLS (QUIC requires TLS 1.3).
3.  **Server Implementation:**
    *   Initialize `quic.Listener`.
    *   Accept incoming streams (for control) and datagrams (for data).
    *   Log received datagrams.
4.  **Client Implementation:**
    *   Dial the server using `quic.DialAddr`.
    *   Send a stream of dummy "Ping" packets using `SendDatagram`.
5.  **Deliverable:**
    *   `cmd/server`: Runs and listens.
    *   `cmd/client`: Connects and floods datagrams.
    *   **Test:** Validate connectivity and ensure datagrams are received (and that packet loss doesn't kill the connection).

## Phase 2: Point-to-Point Tunnel (TUN Integration)
**Goal:** Integrate `water` to create virtual network interfaces (`utun` on macOS, `tun` on Linux) and forward IP packets over QUIC.

1.  **TUN Interface Setup:**
    *   Use `water` library to open a TUN device.
    *   Implement OS-specific IP assignment (using `ifconfig` or `ip` commands via `exec` within the Go code).
2.  **Packet Loop:**
    *   **Read TUN -> Write QUIC:** Read raw IP packets from TUN, wrap them, send as QUIC Datagrams.
    *   **Read QUIC -> Write TUN:** Receive Datagrams, write raw IP packets to TUN.
3.  **Static Addressing:**
    *   Hardcode IPs for now (e.g., Server: `10.100.0.1/24`, Client: `10.100.0.2/24`).
4.  **Deliverable:**
    *   Functional VPN where Client can `ping 10.100.0.1` (Server VIP).
    *   **Test:** Deploy server on Linux (or local), run client on macOS. Verify `ping` works.

## Phase 3: Hub-and-Spoke (Multi-Client Routing)
**Goal:** Enable the server to handle multiple concurrent clients and route traffic intelligently.

1.  **Session Management:**
    *   Server maintains a map: `Virtual IP <-> Active QUIC Session`.
2.  **IP Allocation (Simple IPAM):**
    *   Server dynamically assigns IPs from a pool (e.g., `10.100.0.0/24`) upon handshake.
    *   Communicate assigned IP to client via a reliable QUIC **Stream** (control channel) before starting Datagram exchange.
3.  **Routing Logic:**
    *   When Server reads from TUN, parse destination IP header.
    *   Look up target Session.
    *   Send Datagram to specific session.
4.  **Deliverable:**
    *   Multiple clients can connect.
    *   **Test:** Client A (`.2`) and Client B (`.3`) can both ping Server (`.1`).

## Phase 4: Authentication & Internet Access
**Goal:** Secure the VPN and allow clients to access the internet through the server (Gateway mode).

1.  **Authentication:**
    *   Implement a simple token-based auth or mTLS.
    *   Reject connections in the TLS handshake or immediately after if auth fails.
2.  **Server-Side NAT (Masquerading):**
    *   Enable IP forwarding on the Server OS (`sysctl -w net.ipv4.ip_forward=1`).
    *   Configure `iptables` / `nftables` to MASQUERADE traffic leaving the physical interface coming from the VPN subnet.
3.  **Deliverable:**
    *   Secure VPN.
    *   **Test:** Client works with `curl google.com` routing through the VPN.

## Phase 5: Wails GUI Client
**Goal:** Wrap the macOS CLI client in a user-friendly GUI.

1.  **Wails Setup:** Initialize a Wails project.
2.  **Backend Integration:** Move Client logic into Wails `App` struct.
    *   Expose `Connect(config)` and `Disconnect()` methods to frontend.
    *   Stream logs/stats (upload/download rate) to frontend.
3.  **Frontend:**
    *   Simple React/Vue form for Server Address & Key.
    *   Connect/Disconnect toggle.
    *   Status indicator.
4.  **Deliverable:**
    *   `.app` bundle for macOS.
    *   **Test:** User clicks "Connect", successful connection to Phase 4 Server.

## Phase 6: Cross-Platform & Polish (Future)
*   Windows support (Wintun integration).
*   Linux Desktop support.
*   Performance tuning (MTU discovery, batch reading).
