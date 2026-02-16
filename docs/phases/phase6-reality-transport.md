# Phase 6: Reality-Style Stealth Transport

**Goal:** Implement a transport layer that mimics a legitimate website and resists active probing to bypass aggressive DPI blocking.

## 1. Problem Analysis
The Russian TSPU blocks "Unknown UDP" after a ~30-second grace period. To stay alive, SloPN traffic must be categorized as "Known Good" (e.g., HTTPS/QUIC to a trusted domain).

## 2. Solution: Hybrid First-Packet-Obfuscation (FPO)
We will modify the `obfuscator` package to use a "Temporal Authentication" window.

### Client Tasks:
- [x] **SNI Spoofing**: The client QUIC dialer sends a `ServerName` matching a common domain.
- [ ] **Handshake Obfuscation**: Prepend the 32-byte Magic Header to the first ~10 UDP packets.
- [ ] **Clean Transition**: Automatically drop the obfuscation header once the QUIC handshake is confirmed (or after a fixed packet count), switching to standard QUIC.

### Server Tasks:
- [ ] **Temporal Gatekeeping**: The server inspects initial packets of unknown flows for the Magic Header.
- [ ] **IP Whitelisting**: 
    - Upon receiving a valid FPO packet, the client's IP is added to a "Clean QUIC" whitelist for 1 hour.
    - Subsequent packets from this IP bypass the obfuscator and are fed directly to the QUIC engine as raw datagrams.
- [ ] **Mirroring (Default)**: Packets from non-whitelisted IPs that fail FPO validation are proxied to the mimic target.

## 3. Implementation Steps
1.  **Enhance `pkg/obfuscator`**: Add state tracking to `RealityConn` to manage whitelisted IPs and transition counters.
2.  **Optimize Data Path**: Ensure whitelisted traffic avoids the overhead of HMAC/XOR.
3.  **Update `cmd/client`**: Modify the client loop to handle the transition from FPO to Clean mode.

## 4. Success Criteria
- [ ] Server does not respond to generic `nc -u` probes in a way that reveals it is a VPN.
- [ ] Wireshark identifies the handshake as standard QUIC/TLS to the spoofed domain.
- [ ] Connection remains stable for > 1 hour in restricted environments.
