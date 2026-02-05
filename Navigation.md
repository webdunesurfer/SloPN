# ADR Navigation Map

This document serves as the entry point for understanding the architectural decisions made for the SloPN project. Follow the links below in the recommended reading order to understand the system design.

## 🗺️ Decision Map

### 1. [IP Address Management (IPAM)](ADR-IPAM.md)
**Focus:** How clients get their internal VPN addresses.
- **Decision:** Dynamic, DHCP-like allocation.
- **Why:** Simplicity and efficiency for early-stage development.

### 2. [Authentication Strategy](ADR-AUTHENTICATION.md)
**Focus:** Verifying client identity.
- **Decision:** Token/Key exchange over an encrypted QUIC stream.
- **Why:** Avoids the overhead of managing individual client certificates (mTLS).

### 3. [Control Plane Protocol](ADR-Control-Plane.md)
**Focus:** How the client and server "talk" about configuration.
- **Decision:** JSON over reliable QUIC bidirectional streams.
- **Why:** Human-readable, extensible, and leverages QUIC's built-in reliability.

### 4. [MTU and Fragmentation](ADR-MTU-Fragmentation.md)
**Focus:** Handling packet sizes and network overhead.
- **Decision:** Fixed 1280-byte MTU on the TUN interface.
- **Why:** Avoids complex fragmentation logic while fitting within standard physical network limits.

### 5. [Routing Mode](ADR-Routing.md)
**Focus:** Which traffic goes through the tunnel.
- **Decision:** Full Tunneling (Default Gateway).
- **Why:** Provides maximum privacy and simplest user experience by routing all traffic through the secure server.

---

## 🛠️ Implementation Guidance
When implementing new features or refactoring, refer to [AGENT-Memory.md](AGENT-Memory.md) for the current technical state and the [plan.md](plan.md) for the overall roadmap.
