# Phase 1: The Transport Layer (QUIC Datagrams)

**Goal:** Establish a secure QUIC connection and exchange unreliable datagrams between a CLI client and server. No TUN interfaces yet.

1.  **Setup Project Structure:** Initialize Go module.
2.  **Certificates:** Generate self-signed CA and certificates for TLS (QUIC requires TLS 1.3).
3.  **Server Implementation:**
    *   Initialize `quic.Listener`.
    *   Accept incoming streams (for control) and datagrams (for data).
    *   Log received datagrams.
4.  **Client Implementation:**
    *   Dial the server using `quic.DialAddr`.
    *   Send a stream of dummy "Ping" packets using `SendDatagram`.
5.  **Deliverable:**
    *   `cmd/server`: Runs and listens.
    *   `cmd/client`: Connects and floods datagrams.
    *   **Test:** Validate connectivity and ensure datagrams are received (and that packet loss doesn't kill the connection).
