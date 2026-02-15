# Phase 6: Reality-Style Stealth Transport

**Goal:** Implement a transport layer that mimics a legitimate website and resists active probing to bypass aggressive DPI blocking.

## 1. Problem Analysis
The Russian TSPU blocks "Unknown UDP" after a ~30-second grace period. To stay alive, SloPN traffic must be categorized as "Known Good" (e.g., HTTPS/QUIC to a trusted domain).

## 2. Solution: The "Reality" Wrapper
We will modify the `obfuscator` package to move from simple XOR to a "Pre-Handshake Validator".

### Client Tasks:
- [ ] Implement **SNI Spoofing**: The client QUIC dialer will send a `ServerName` matching a common domain (e.g., `www.microsoft.com`).
- [ ] Implement **The Magic Packet**: Prepend a small, encrypted header (using the `SLOPN_TOKEN`) to the first QUIC packet.
- [ ] **Jitter/Padding**: Randomize initial packet sizes to match the target domain's typical handshake size.

### Server Tasks:
- [ ] **Stateful Inspection**: The server UDP listener will hold new flows in a "Pending" state.
- [ ] **Secret Validation**: 
    - If the packet decrypts with `SLOPN_TOKEN`, the flow is promoted to the QUIC engine.
    - If validation fails, the server responds with a legitimate-looking TLS alert or mimics a real server's response (Mirroring).
- [ ] **Dynamic IP Whitelisting**: Once a client is authenticated, its IP is whitelisted for the duration of the session to reduce CPU overhead on subsequent packets.

## 3. Implementation Steps
1.  **Refactor `pkg/obfuscator`**: Create a `RealityConn` that handles the flow-splitting logic.
2.  **Update `cmd/server`**: Configure a "Dest" target (e.g., `13.107.4.52:443` for Microsoft) to mirror for invalid probes.
3.  **Update `cmd/helper`**: Add `Reality` mode toggle and target domain configuration.

## 4. Success Criteria
- [ ] Server does not respond to generic `nc -u` probes in a way that reveals it is a VPN.
- [ ] Wireshark identifies the handshake as standard QUIC/TLS to the spoofed domain.
- [ ] Connection remains stable for > 1 hour in restricted environments.
