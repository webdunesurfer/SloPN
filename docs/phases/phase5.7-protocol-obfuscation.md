# Phase 5.7: Protocol Obfuscation (Stealth Mode)

**Goal:** Make SloPN traffic invisible to Deep Packet Inspection (DPI) systems by masking the identifiable QUIC headers and disrupting packet size analysis.

## 1. The Problem
Standard QUIC (RFC 9000) packets have a cleartext "Header" in the first few bytes (Flags, Version, Connection ID). DPI systems can easily identify this pattern and block the connection, even if the payload is encrypted.

## 2. The Solution: "The Onion Wrapper"
We will implement a middleware layer (`net.PacketConn`) that sits between the OS network stack and the `quic-go` engine. This wrapper will transform every UDP packet before it hits the wire.

### Core Strategy: XOR Scrambling + Padding
Instead of "Double Encryption" (which is slow), we will use a high-performance **XOR Scramble** seeded by the user's `Authentication Token`.

1.  **Header Masking:** The entire UDP payload (including the QUIC header) is XORed with a keystream derived from the session token.
    *   *Result:* The "QUIC Bit" and Version fields disappear. The packet looks like high-entropy random noise.
2.  **Junk Padding (Optional):** We will randomly append 0-16 bytes of garbage to the end of packets.
    *   *Result:* This disrupts "Fingerprinting" based on exact packet sizes (e.g., distinguishing a standard TLS ClientHello from a generic data packet).

## 3. Implementation Plan

### A. New Package: `pkg/obfuscator`
*   Create a struct `ObfuscatedConn` that implements `net.PacketConn`.
*   **Write Path (`WriteTo`):**
    1.  Generate a unique "IV" (Initialization Vector) or use a rolling counter.
    2.  Mask the payload using the Token + IV.
    3.  Prepend the IV (if needed) or rely on implicit state.
    4.  Send the scrambled bytes to the real UDP socket.
*   **Read Path (`ReadFrom`):**
    1.  Read scrambled bytes from the real UDP socket.
    2.  Unmask the payload using the Token.
    3.  Pass the clean QUIC packet up to `quic-go`.

### B. Integration
*   **Server (`cmd/server`):**
    *   Add flag `-obfs=true`.
    *   Wrap the `udp` listener with `obfuscator.ListenPacket`.
*   **Helper (`cmd/helper`):**
    *   Update the protocol to support an "Obfuscated Dial" mode.
    *   The `LoginRequest` is inside the QUIC stream, so the obfuscation key must be derived from the **Token** (which the user already has) *before* the handshake starts.

## 4. Architectural Decision (ADR)
See **[ADR-Obfuscation](../adr/ADR-Obfuscation.md)**.

We have adopted **Path A: Shared Transport Secret**.
*   The `SLOPN_TOKEN` will serve as the global **Transport Secret**.
*   **Key Derivation:** We will use `HKDF-SHA256` to derive the actual XOR masking key from this token. This ensures cryptographic strength even if the user provides a simple string.
*   **Implication:** For now, the "Door Key" (Obfuscation) and "Session Key" (Authentication) are derived from the same source. This allows the server to descramble incoming packets immediately without knowing the user's identity first.

## 5. Success Criteria
*   [ ] Wireshark/tcpdump shows "Malformed Packet" or "UDP Data" instead of "QUIC".
*   [ ] Connection establishes successfully with `-obfs` enabled.
*   [ ] Performance impact is negligible (< 1% CPU overhead).
