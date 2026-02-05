# Phase 3.2: Multi-Client Routing & Inter-Client Communication

**Goal:** Verify that the server can manage multiple concurrent client sessions, assign unique IPs, and route traffic between clients (Spoke-to-Spoke).

## Objectives
1.  **Concurrent Connectivity:** Ensure two or more clients can stay connected without session collisions.
2.  **Server Reachability:** Every client must be able to ping the Server VIP (`10.100.0.1`).
3.  **Client-to-Client Routing:** Client A (`10.100.0.2`) must be able to ping Client B (`10.100.0.3`) through the server.

## Challenges (Single Machine Testing)
*   **Routing Conflicts:** Multiple clients on one macOS machine will compete for the `10.100.0.0/24` route.
*   **Interface Selection:** We must ensure the `ping` command uses the correct `utun` interface during tests.

## Implementation Steps

### 1. Verification of Spoke-to-Spoke Routing
*   The server's `TUN -> QUIC` loop already checks `sm.GetSession(destIP)`.
*   We need to ensure that when Client A sends a packet to Client B, the server receives it from QUIC, writes it to its own TUN, the kernel "loops it back" to the TUN, and the server then reads it and forwards it to Client B.
*   *Alternatively:* Optimize by checking the destination IP directly in the `QUIC -> TUN` receive loop to see if it's another client, bypassing the local kernel stack for inter-client traffic.

### 2. Multi-Client Test Script
*   Create a test setup that launches two client processes with different configuration files.
*   Use `ping -S <source_ip>` to force the OS to use the correct virtual interface for each client.

## Deliverables
*   Logged proof of Client A (`.2`) pinging Client B (`.3`).
*   Verified IPAM stability (IPs are released and reused correctly).
