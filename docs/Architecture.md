# SloPN Architecture

SloPN (Slow Private Network) is a QUIC-based Layer 3 VPN designed for simplicity, security, and performance.

## Core Protocols
- **QUIC (RFC 9000):** Used as the primary transport layer. QUIC provides the reliability of TCP for control signals and the speed of UDP for data transfer.
- **TLS 1.3:** Built into QUIC, ensuring all traffic between the client and server is encrypted by default.
- **Layer 3 (IP):** The VPN operates at the IP layer, tunneling raw IPv4 packets over QUIC Datagrams.

## Data Flow
1. **Control Plane:** A standard QUIC stream is opened upon connection for the Login handshake (JSON-based protocol).
2. **Data Plane:** Once authenticated, raw IP packets are intercepted by a TUN interface, wrapped in **QUIC Datagrams** (unreliable/out-of-order, ideal for VPNs), and sent to the peer.
3. **Server Routing:** The server acts as a gateway, using a Session Manager to route packets between clients or NATing them to the public internet.

## Security & Encryption
- **Encryption:** All traffic is encrypted using TLS 1.3 via QUIC. Currently uses self-signed certificates (Phase 4).
- **Authentication:** Token-based authentication. The client must provide a matching token in the initial JSON handshake.
- **Integrity:** QUIC provides built-in integrity checks for every packet, preventing tampering.

## Client Isolation
- Each client is assigned a unique Virtual IP (VIP) from the `10.100.0.0/24` subnet.
- The server's Session Manager maps VIPs to specific QUIC connections.
- **Isolation Level:** Clients can currently talk to each other if the server's "Spoke-to-Spoke" fast path is enabled, but they are isolated from the server's local network unless explicitly routed.

## GUI & Helper Architecture
Starting from Phase 5, the client is split into two components to handle macOS/Linux security models:
- **Privileged Helper (Engine):** Runs as `root` to manage TUN interfaces and system routing. Listens on a local TCP port (`127.0.0.1:54321`) for IPC commands.
- **GUI Dashboard (Wails):** Runs in user space, providing a Svelte-based interface for controlling the helper.

## IPC Mechanism
- **TCP Bridge:** Communication between the GUI and Helper uses a local TCP socket. This bypasses macOS sandbox restrictions that often block Unix Domain Sockets for App Bundles.
- **JSON Protocol:** Commands (`connect`, `disconnect`, `status`) and real-time statistics are exchanged as JSON objects.

## OS Specifics
### macOS
- **Helper:** Uses `/dev/tunX` and manual `route` management.
- **GUI:** Wails-based Svelte application.
- **IPC:** TCP port 54321.

### Linux (Server)
- **Forwarding:** Uses `sysctl` for `net.ipv4.ip_forward`.
- **NAT:** Uses `iptables` MASQUERADE for internet exit.

## Component Overview
- **`pkg/protocol`:** QUIC Handshake messages.
- **`pkg/ipc`:** Inter-Process Communication messages between GUI and Helper.
- **`pkg/tunutil`:** TUN interface abstraction.
- **`pkg/session`:** Server-side connection management.
- **`pkg/iputil`:** IP header manipulation and packet summaries.
