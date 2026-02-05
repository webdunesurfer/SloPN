# SloPN: Custom QUIC-based VPN

SloPN is a modern "Hub-and-Spoke" VPN solution designed for high performance and security. It leverages the **QUIC** protocol and **RFC 9221 (Unreliable Datagrams)** to provide a robust tunneling experience while avoiding the common "TCP-over-TCP" performance issues found in traditional VPNs.

## 🚀 Vision
- **Secure by Default:** TLS 1.3 encryption via QUIC.
- **High Performance:** Unreliable datagrams for low-latency packet forwarding.
- **Simple Management:** Hub-and-spoke architecture with dynamic IP assignment.
- **User Friendly:** Cross-platform GUI built with Wails.

## 🛠 Tech Stack
- **Language:** [Go (Golang)](https://go.dev/)
- **Protocol:** [quic-go](https://github.com/quic-go/quic-go) (RFC 9221)
- **Tunneling:** [`water`](https://github.com/songgao/water) (TUN/TAP)
- **GUI:** [Wails](https://wails.io/) (Go + React/Vue)

## 📖 Navigation for Agents & Developers
If you are an LLM agent or a new contributor, please refer to these documents in order:

1.  **[Navigation.md](docs/Navigation.md):** The master map of all Architectural Decision Records (ADRs). Start here to understand the core design decisions (IPAM, Auth, MTU, Routing).
2.  **[plan.md](docs/plan.md):** Our multi-phase roadmap from transport layer to final GUI.
3.  **[AGENT-Memory.md](docs/AGENT-Memory.md):** A living document capturing the current technical state, environment details, and immediate next steps.

## 🛣 Roadmap
- **Phase 1:** QUIC Transport & Datagram Exchange (Current)
- **Phase 2:** TUN Interface Integration
- **Phase 3:** Multi-client Routing & IPAM
- **Phase 4:** Authentication & Internet Gateway
- **Phase 5:** Wails GUI implementation

---
*Maintained by webdunesurfer*
