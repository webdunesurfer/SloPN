# Phase 2: Point-to-Point Tunnel (TUN Integration)

**Goal:** Integrate `water` to create virtual network interfaces (`utun` on macOS, `tun` on Linux) and forward IP packets over QUIC.

1.  **TUN Interface Setup:**
    *   Use `water` library to open a TUN device.
    *   Implement OS-specific IP assignment (using `ifconfig` or `ip` commands via `exec` within the Go code).
2.  **Packet Loop:**
    *   **Read TUN -> Write QUIC:** Read raw IP packets from TUN, wrap them, send as QUIC Datagrams.
    *   **Read QUIC -> Write TUN:** Receive Datagrams, write raw IP packets to TUN.
3.  **Static Addressing:**
    *   Hardcode IPs for now (e.g., Server: `10.100.0.1/24`, Client: `10.100.0.2/24`).
4.  **Deliverable:**
    *   Functional VPN where Client can `ping 10.100.0.1` (Server VIP).
    *   **Test:** Deploy server on Linux (or local), run client on macOS. Verify `ping` works.
