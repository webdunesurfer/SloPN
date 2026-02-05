# ADR: Routing Mode

## Status
Approved

## Context
The VPN needs to decide which traffic from the client machine should be routed through the secure tunnel. This affects privacy, performance, and the complexity of the client-side network configuration.

## Decision
We will implement **Full Tunneling** as the default routing mode.

1.  **Client Behavior:** Upon successful connection and TUN interface initialization, the client will modify the system routing table to set the VPN's Virtual IP (Server VIP) as the default gateway.
2.  **Server Behavior:**
    *   The server must have IP forwarding enabled (`net.ipv4.ip_forward=1`).
    *   The server will perform Network Address Translation (NAT/Masquerading) for packets originating from the VPN subnet and exiting via its physical internet interface.
3.  **DNS:** To prevent DNS leaks, the client should ideally point the system's DNS to a trusted resolver (or the Server VIP) while connected.

## Consequences
*   **Pros:**
    *   **Maximum Privacy:** All internet traffic is encrypted and hidden from the client's local ISP.
    *   **Simplicity:** No need to manage complex "included/excluded" IP lists for specific services.
    *   **Bypassing Restrictions:** Effectively bypasses local network censorship or geoblocking for all applications.
*   **Cons:**
    *   **Latency:** All traffic (even local-to-local if not carefully handled) may be routed through the remote server.
    *   **Server Load:** The server must handle the bandwidth requirements for all client activities (e.g., video streaming).
    *   **Complexity:** Modifying the default gateway is a "high-privilege" operation and can be brittle if the connection drops unexpectedly.
