# ADR: Control Plane Protocol

## Status
Approved

## Context
The client and server need a reliable mechanism to exchange configuration data, such as authentication tokens, assigned virtual IPs, and routing information. This "Control Plane" must be separate from the "Data Plane" (which uses unreliable datagrams).

## Decision
We will use **JSON-encoded messages** sent over a **Reliable QUIC Bidirectional Stream**.

1.  **Protocol:** Immediately after the QUIC connection is established, the client opens a stream. Both parties will use this stream for a request-response handshake.
2.  **Message Format:** Each message will be a JSON object followed by a newline (Line-delimited JSON) to simplify parsing.
3.  **Extensibility:** JSON allows us to add new fields (e.g., DNS settings, health stats) without breaking compatibility during the early stages of development.

### Proposed Message Structures

#### 1. Client Login (Client -> Server)
```json
{
  "type": "login_request",
  "token": "secret-auth-token",
  "client_version": "0.1.0",
  "os": "macos"
}
```

#### 2. Login Response (Server -> Client)
```json
{
  "type": "login_response",
  "status": "success",
  "assigned_vip": "10.100.0.2",
  "subnet_mask": "255.255.255.0",
  "server_vip": "10.100.0.1",
  "message": "Welcome to SloPN"
}
```

#### 3. Heartbeat / Keepalive (Optional/Periodic)
```json
{
  "type": "heartbeat",
  "timestamp": 1700000000
}
```

## Consequences
*   **Pros:**
    *   **Human Readable:** Easy to debug using packet captures or logs.
    *   **Flexible:** Adding optional features later is trivial.
    *   **Go Support:** Native `encoding/json` is robust and easy to use.
*   **Cons:**
    *   **Verbosity:** Higher overhead compared to binary formats like Protobuf, but negligible for control plane traffic (which is low frequency).
