# Phase 4: Authentication & Internet Access

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
