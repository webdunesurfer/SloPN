# ADR: Reality-Style UDP Obfuscation

## Status
Accepted

## Context
Our previous XOR-based obfuscation (Phase 5.7) is insufficient against the state-of-the-art Deep Packet Inspection (DPI) used in high-censorship regions (like Russia). Specifically, the "30-second drop" behavior suggests that "Unknown UDP" flows are being flagged and terminated after a short window. To bypass this, we need more than just encryption; we need **mimicry** that can survive **Active Probing**.

## Decision
We will implement a **"Reality-style" UDP transport** (inspired by VLESS-Reality) with a hybrid **First-Packet-Obfuscation (FPO)** mechanism.

### Key Mechanics:
1.  **Identity Theft (Mirroring):** The SloPN server will "borrow" the TLS/QUIC identity of a legitimate, high-traffic website (e.g., `www.google.com` or `microsoft.com`).
2.  **First-Packet-Obfuscation (FPO):** 
    - The client obfuscates *only* the initial handshake packets (e.g., first 10-20 packets) by prepending a 32-byte Magic Header (Salt + HMAC) and XORing the payload.
    - This leverages the "Grace Period" observed in Russian DPI (TSPU), where unknown UDP is permitted for approximately 10-30 seconds or 50-100 packets.
3.  **Clean-QUIC Transition:**
    - Once the server validates the Magic Header, it whitelists the client's IP and expects standard, non-obfuscated QUIC packets for the remainder of the session.
    - Standard QUIC matches the "Known Good" fingerprints of services like YouTube or Microsoft telemetry, which are allowed indefinitely.
4.  **Active Probing Resistance:** If the server receives a probe that lacks the Magic Header (or is sent after the FPO window without prior authorization), it acts as a transparent mirror for the target website.

## Implementation Path
- **Client:** Wraps initial QUIC packets in the FPO header, then switches to a "Clean Mode" raw UDP socket.
- **Server:** Tracks "Authorized" IPs. 
  - **Authorized Flow:** Forwards raw UDP directly to the QUIC engine.
  - **New/Unauthorized Flow:** Inspects for the Magic Header. If valid, promotes to Authorized. If invalid, proxies to the mimic target.

## Consequences
- **Security:** Defeats active probing. The censor cannot prove the server is a VPN because it appears as a legitimate site to both probes and long-lived flow analyzers.
- **Reliability:** Bypasses the "Unknown UDP" drop-off by blending into "Known QUIC" traffic once the session is established.
- **Performance:** Eliminates the 32-byte overhead and XOR CPU cost for 99.9% of the session traffic.
