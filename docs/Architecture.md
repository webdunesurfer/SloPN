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

## OS Specifics
### macOS (Client)
- Uses `/dev/tunX` via the `water` library.
- Routing is managed via the `route` command (e.g., `route add default ...`).
- Gateway detection uses `route -n get default`.

### Linux (Server)
- Uses `tuntap` interfaces.
- Uses `sysctl` to enable `ip_forward`.
- Uses `iptables` with `MASQUERADE` for NAT/Internet access.
- Pre-creates interfaces with `nopi` (No Packet Information) to match Linux kernel expectations.

## Component Overview
- **`pkg/protocol`:** Defines the JSON handshake messages.
- **`pkg/tunutil`:** Cross-platform abstraction for creating and configuring TUN interfaces.
- **`pkg/session`:** Server-side logic for IP allocation and connection tracking.
- **`pkg/iputil`:** Helpers for parsing raw IP headers and adding/stripping OS-specific headers.
