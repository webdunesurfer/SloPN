# Phase 3: Hub-and-Spoke (Multi-Client Routing)

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
