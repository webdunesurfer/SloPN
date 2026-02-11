# ADR: Protocol Obfuscation Strategy

## Context
SloPN uses QUIC (RFC 9000) for its transport layer. While performant, standard QUIC packets have a cleartext header (Flags, Version, Connection ID) that makes them easily identifiable by Deep Packet Inspection (DPI) systems. To resist active probing and censorship, we need to mask this "wire signature."

## The Problem: Identity vs. Transport
We want to scramble the UDP packets using a secret key. However, the server needs to know *which* key to use to descramble the packet *before* it can read the packet's content (which contains the user's identity/token).

*   **Scenario:** If every user has a unique token, the server receives a blob of random bytes and doesn't know which user sent it, making it impossible to pick the correct decryption key.

## Decision: Shared Transport Secret
We will adopt a **"Transport Secret"** model, similar to WireGuard's Pre-Shared Key (PSK) or a ShadowSocks server password.

1.  **Single Shared Secret:** The Obfuscation Layer will use a **Global Shared Secret** derived from the server's configured `SLOPN_TOKEN`.
2.  **Scope:** This secret is used *only* to mask the UDP headers and payload to bypass DPI. It is **NOT** the primary authentication mechanism for user sessions (which remains inside the TLS 1.3 tunnel).
3.  **Key Derivation:** We will use **HKDF-SHA256** to derive a specific "Obfuscation Key" from the input Token. This ensures that even if the Token is short/weak, the XOR keystream has better statistical properties.

## Implications
*   **Forward Compatibility:** If/when we move to a multi-user database (User A, User B), all users must still share this one "Gatekeeper Key" to establish the initial connection. Individual authentication happens *inside* the obfuscated tunnel.
*   **Security:** This does not replace TLS. It effectively adds a "Door Code" to the UDP port.
*   **Performance:** XOR masking is extremely fast and negligible compared to the inner AES/ChaCha encryption.

## Status
Accepted for Phase 5.7.
