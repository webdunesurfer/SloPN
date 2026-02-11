# ADR: Authentication Strategy

## Status
Approved

## Context
The VPN requires a secure way to verify client identity before allowing traffic to flow through the tunnel. Since we are using QUIC, we already have mandatory TLS 1.3 encryption.

## Decision
We will use **Token-Based Authentication** over a established QUIC connection.

1.  **Single Server Certificate:** The server will use a single TLS certificate (self-signed or CA-issued) to establish the encrypted QUIC tunnel.
2.  **Handshake Completion:** The QUIC connection will be established first, ensuring all subsequent communication is encrypted.
3.  **Control Stream Login:** Immediately after connection, the client must open a reliable QUIC bidirectional stream and send a "Login" message containing a pre-shared token or unique key.
4.  **Verification:** The server will validate the token. 
    *   If valid, the server sends a "Login Success" message along with the assigned Virtual IP.
    *   If invalid, the server closes the QUIC connection immediately.
5.  **Traffic Gatekeeping:** No Datagrams (VPN traffic) will be processed by the server for a session until the authentication on the control stream is successful.

## Consequences
*   **Pros:**
    *   Easier to manage than mTLS (no need to issue/revoke unique certificates for every client).
    *   Flexible: The "Token" can be a simple string, a JWT, or a database-backed API key.
    *   Decouples transport security (TLS) from identity (Token).
    *   **Dual Use:** As of Phase 5.7, this Token also seeds the UDP Obfuscation Layer (see [ADR-Obfuscation](ADR-Obfuscation.md)), allowing pre-handshake packet masking.
*   **Cons:**
    *   Requires a custom application-level handshake after the QUIC handshake.
    *   The server is theoretically exposed to the "Login" message from any client that trusts the server's certificate.
