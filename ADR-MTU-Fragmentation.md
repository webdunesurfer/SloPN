# ADR: MTU and Fragmentation

## Status
Approved

## Context
Encapsulating IP packets within QUIC Datagrams adds overhead (IP + UDP + QUIC headers). If the resulting packet exceeds the Path MTU (typically 1500 bytes on Ethernet), it will be fragmented or dropped by the physical network. `quic-go` Datagrams do not automatically handle fragmentation of payload.

## Decision
We will use a **Fixed Lower MTU** on the Virtual Interface (TUN) and **Avoid Fragmentation Logic**.

1.  **MTU Setting:** The TUN interface on both client and server will be configured with an MTU of **1280 bytes**.
2.  **Rationale for 1280:** 
    *   1280 is the minimum MTU required for IPv6, making it a safe baseline for most networks.
    *   With a standard 1500-byte physical MTU, this provides 220 bytes of "headroom" for the outer headers (UDP, IP, and QUIC Datagram overhead).
3.  **No Fragmentation:** The application will not implement logic to split large IP packets into multiple QUIC Datagrams. Packets arriving at the TUN interface that exceed 1280 bytes will be dropped by the OS or ignored by our logic.
4.  **MSS Clamping (Future):** If needed, we will implement TCP MSS Clamping to ensure TCP sessions automatically negotiate segments that fit within the 1280-byte MTU.

## Consequences
*   **Pros:**
    *   Significantly simplifies the implementation of the packet loop.
    *   Reduces CPU overhead on the server/client by avoiding assembly/disassembly of fragments.
    *   High compatibility with restricted physical networks (e.g., mobile, satellite).
*   **Cons:**
    *   Slightly lower throughput efficiency due to higher header-to-payload ratio.
    *   May cause issues with certain UDP-based protocols that do not support Path MTU Discovery (PMTUD), though 1280 is generally very safe.
