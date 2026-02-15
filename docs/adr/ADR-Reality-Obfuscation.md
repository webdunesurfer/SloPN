# ADR: Reality-Style UDP Obfuscation

## Status
Accepted

## Context
Our previous XOR-based obfuscation (Phase 5.7) is insufficient against the state-of-the-art Deep Packet Inspection (DPI) used in high-censorship regions (like Russia). Specifically, the "30-second drop" behavior suggests that "Unknown UDP" flows are being flagged and terminated after a short window. To bypass this, we need more than just encryption; we need **mimicry** that can survive **Active Probing**.

## Decision
We will implement a **"Reality-style" UDP transport** (inspired by VLESS-Reality). 

### Key Mechanics:
1.  **Identity Theft (Mirroring):** The SloPN server will "borrow" the TLS/QUIC identity of a legitimate, high-traffic website (e.g., `microsoft.com` or `yandex.ru`). 
2.  **Stealth Handshake:** The initial UDP packet from the client will look like a standard QUIC ClientHello for the target site.
3.  **Active Probing Resistance:** If the server receives a probe from a censor (a generic "Hello" or malformed QUIC packet), it will respond exactly as the target website would (or silently drop/forward the probe).
4.  **The Magic Secret:** The server will only "unlock" the VPN tunnel when it receives a specific, cryptographically signed payload within the spoofed ClientHello that matches the server's `SLOPN_TOKEN`.

## Implementation Path
- **Client:** Must wrap its initial QUIC handshake in a packet that mimics the target site's TLS signature.
- **Server:** Acts as a "Gatekeeper." It inspects the first packet of every new UDP flow.
  - If it contains the **Magic Secret**: It treats the flow as a SloPN session.
  - If it does **NOT**: It acts as a transparent proxy or mirror for the target site, making it indistinguishable from a real server.

## Consequences
- **Security:** Defeats active probing. The censor cannot prove the server is a VPN without the secret token.
- **Reliability:** Since it mimics a "must-have" service (like Microsoft/Google/Yandex infrastructure), it is much less likely to be blocked.
- **Complexity:** Requires a significant refactor of the UDP listener to support "Pre-Handshake Inspection."
