# Phase 3.1: Reliable ICMP & Routing

**Goal:** Ensure 100% reliable ICMP connectivity between Client <-> Server and Client <-> Client. Resolve Linux/macOS kernel routing quirks.

## Analysis of Current Issues
1.  **Server Ingress Silent Drop:** When a packet from a client (e.g., `10.100.0.2`) reaches the server app and is written to `tun0`, the Linux kernel may drop it if it doesn't consider the source IP "valid" for that interface (Reverse Path Filtering) or if it doesn't recognize the destination as itself.
2.  **MTU/Fragmentation:** Although we set MTU to 1280, we need to ensure the Go buffers and the QUIC transport are not causing silent truncations.
3.  **macOS PTP Topology:** macOS `utun` behaves differently than Linux `tun`. We need to ensure the routing table and interface configuration are perfectly aligned.

## Implementation Steps

### 1. Advanced Debugging (IP Packet Inspection)
*   Enhance `pkg/iputil` to parse ICMP types (Request vs Reply).
*   Add verbose logging to both server and client to show: `Direction | Source -> Dest | Protocol | Type`.

### 2. Linux Kernel Tuning (Server)
*   Explicitly set `accept_local=1` and `rp_filter=0` via `sysctl`.
*   Verify the `tun0` configuration: ensure it is a point-to-point link when only one client is present, or a proper subnet when multiple are.

### 3. macOS Routing Fixes (Client)
*   Ensure the `utun` interface is configured with a proper peer IP.
*   Fix the `route add` logic to be more resilient to existing routes.

### 4. ICMP "Loopback" Test
*   Implement a test where the server can ping itself via `10.100.0.1` and the client can ping itself via its assigned VIP.

## Deliverables
*   Successful `ping 10.100.0.1` from Client.
*   Successful `ping 10.100.0.2` from Server.
*   Successful `ping 10.100.0.3` from Client A to Client B (Inter-client routing).
