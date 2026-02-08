# SloPN Architecture

SloPN (Slow Private Network) is a QUIC-based Layer 3 VPN designed for simplicity, security, and performance.

## Core Protocols
- **QUIC (RFC 9000):** Primary transport for control and data.
- **DNS-over-Tunnel:** All DNS queries are routed to the **SloPN Internal DNS (`10.100.0.1`)** which uses a self-hosted CoreDNS instance. This provides metadata privacy by hiding browsing history from the ISP.
- **TLS 1.3:** Built-in encryption for all tunnel traffic.

## Data Flow
1. **Control Plane:** A standard QUIC stream is opened upon connection for the Login handshake (JSON-based protocol).
2. **Data Plane:** Once authenticated, raw IP packets are intercepted by a TUN interface, wrapped in **QUIC Datagrams** (unreliable/out-of-order, ideal for VPNs), and sent to the peer.
3. **Server Routing:** The server acts as a gateway, using a Session Manager to route packets between clients or NATing them to the public internet.

## DNS Architecture
To ensure complete metadata privacy and prevent leaks, SloPN implements a self-hosted DNS infrastructure:
- **Server-Side:** A **CoreDNS** container runs alongside the VPN server. It is configured as a recursive resolver that forwards queries to upstream root servers (Google/Cloudflare) while maintaining a local cache.
- **Redirection:** The SloPN server uses `iptables` DNAT rules to intercept all traffic on port 53 (UDP/TCP) coming from the `tun0` interface and redirects it to the host's Docker Bridge IP where CoreDNS is listening.
- **Client-Side:** The Helper automatically configures the system's DNS settings to point to the Server VIP (`10.100.0.1`) when Full Tunneling is active.

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
- **Native Bridge:** On macOS, a custom CGO/Objective-C bridge manages the system tray icon to avoid conflicts with the Wails main loop.

## IPC Mechanism
- **TCP Bridge:** Communication between the GUI and Helper uses a local TCP socket. This bypasses macOS sandbox restrictions that often block Unix Domain Sockets for App Bundles.
- **Security:** Starting from v0.2.2, all IPC requests are authenticated using a **Shared Secret** generated during installation (`/Library/Application Support/SloPN/ipc.secret`). The Helper rejects any requests that do not include the correct secret.
- **JSON Protocol:** Commands (`connect`, `disconnect`, `status`, `get_logs`) and real-time statistics are exchanged as JSON objects.

## OS Specifics
### macOS
- **Helper:** Uses `/dev/tunX` and manual `route` management.
- **DNS Leak Protection:** Starting from v0.2.6, the Helper forces the system to use the **SloPN Internal DNS (`10.100.0.1`)** when Full Tunnel is enabled. This ensures that DNS queries never leave the encrypted tunnel and are resolved directly by the SloPN server.
- **GUI:** Wails-based Svelte application with custom native `NSStatusItem` bridge.
- **IPC:** TCP port 54321.

### Linux (Server)
- **Forwarding:** Uses `sysctl` for `net.ipv4.ip_forward`.
- **NAT:** Uses `iptables` MASQUERADE for internet exit.
- **Rate Limiting:** Starting from v0.2.3, the server implements application-level brute-force protection. It tracks failed authentication attempts per IP and automatically bans malicious IPs for a configurable duration.
- **Dockerization:** The server is packaged as a multi-stage Docker image (Debian-slim base).

## Secure Configuration Storage
Starting from v0.2.1, the client implements a dual-layer persistence strategy for improved security and reliability:
- **Sensitive Data (Auth Tokens):** Stored securely in the platform's native secure storage (e.g., **macOS Keychain**) using the `go-keyring` library. This ensures that secret tokens are encrypted at rest and never exposed in plain text configuration files.
- **Non-Sensitive Data (Server Address, Tunnel Mode):** Persisted in a structured JSON file at `~/Library/Application Support/SloPN/settings.json`. This provides reliable state management that persists across application updates and cache clears.

## Component Overview
- **`pkg/protocol`:** QUIC Handshake messages.
- **`pkg/ipc`:** Inter-Process Communication messages between GUI and Helper.
- **`pkg/tunutil`:** TUN interface abstraction.
- **`pkg/session`:** Server-side connection management.
- **`pkg/iputil`:** IP header manipulation and packet summaries.
