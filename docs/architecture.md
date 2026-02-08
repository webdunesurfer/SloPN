# SloPN Architecture

SloPN (Slow Private Network) is a modular, high-security Layer 3 VPN built with Go and QUIC.

## Core Protocols
- **QUIC (RFC 9000):** Primary transport for control and data. It provides the reliability of TCP for signaling and the performance of UDP for tunneling.
- **TLS 1.3:** Built into QUIC, ensuring all traffic is encrypted and authenticated by default.
- **Layer 3 (IP):** The VPN tunnels raw IPv4 packets over QUIC Datagrams (RFC 9221).

## Data Flow
1. **Control Plane:** A reliable QUIC stream is used for the authenticated Login handshake (JSON-based).
2. **Data Plane:** Once authenticated, raw IP packets are intercepted by a virtual TUN interface, wrapped in unreliable QUIC Datagrams, and forwarded to the peer.
3. **Server Routing:** The server acts as a hub, using a Session Manager to route packets between clients or NATing them to the public internet.

## DNS Architecture
To ensure complete metadata privacy and prevent leaks, SloPN implements a self-hosted DNS infrastructure:
- **Server-Side:** A **CoreDNS** container runs alongside the VPN server as a recursive resolver with a local cache.
- **Redirection:** The server uses `iptables` DNAT rules to intercept traffic on port 53 (UDP/TCP) coming from the `tun0` interface and redirects it to the host's Docker Bridge IP where CoreDNS is listening.
- **Client-Side:** The Helper automatically configures the system's DNS settings to point to the Server VIP (`10.100.0.1`) when Full Tunneling is active.

## Security & Encryption
- **Encryption:** All tunnel traffic is encrypted using TLS 1.3 via QUIC.
- **Authentication:** Token-based authentication. Clients must provide a secure 32-character hex token generated during server installation.
- **IPC Security:** GUI-to-Helper communication is secured via a unique **Shared Secret** generated during installation.
- **Secure Storage:** Sensitive tokens are stored in the platform's native secure storage (e.g., **macOS Keychain**).

## OS Specifics
### macOS (Client)
- **Helper (Engine):** Privileged background service that manages `/dev/tunX` and system routing.
- **GUI Dashboard:** Wails-based Svelte application with custom native `NSStatusItem` bridge for the menu bar.
- **DNS Protection:** Automatic backup and restoration of local DNS settings.

### Linux (Server)
- **Containerization:** The server is deployed via Docker with `NET_ADMIN` capabilities.
- **Rate Limiting:** Application-level brute-force protection that automatically bans malicious IPs.
- **NAT:** Uses `iptables` MASQUERADE for transparent internet exit.

## Component Overview
- **`pkg/protocol`:** QUIC Handshake and control messages.
- **`pkg/ipc`:** Inter-Process Communication between GUI and Helper.
- **`pkg/tunutil`:** TUN interface abstraction and OS-specific configuration.
- **`pkg/session`:** Server-side session management and IPAM.
- **`pkg/iputil`:** IP header manipulation and packet inspection.