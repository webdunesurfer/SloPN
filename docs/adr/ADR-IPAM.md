# ADR: IP Address Management (IPAM)

## Status
Approved

## Context
The VPN follows a Hub-and-Spoke architecture where multiple clients connect to a central server. Each client needs a unique Virtual IP (VIP) within a private subnet to enable routing and communication.

## Decision
We will implement a **Dynamic IP Allocation** strategy (DHCP-like) managed by the server.

1.  **Server-Side Pool:** The server will be configured with a CIDR range (e.g., `10.100.0.0/24`).
2.  **Volatile Leases:** IP addresses will be assigned to clients upon a successful handshake.
3.  **No Persistence:** The server will not remember IP assignments between sessions. If a client disconnects and reconnects, it may receive a different IP from the available pool.
4.  **In-Memory Management:** The server will maintain an in-memory map or bitmask of the subnet to track which IPs are currently "leased" to active QUIC sessions.

## Consequences
*   **Pros:** 
    *   Simplified server logic (no database or persistent storage required for IP mapping).
    *   Efficient use of the IP pool.
*   **Cons:** 
    *   Clients cannot rely on having a static "Internal IP" for long-term services (though they can still be reached via the Server's fixed VIP).
    *   Reconnection events will trigger a change in the client's virtual interface configuration.
