# ADR-GUI-IPC: Communication between GUI and Privileged Helper

## Status
Accepted

## Context
To provide a "one-click" VPN experience on macOS without requiring a `sudo` password for every connection, the SloPN client architecture is split into two components:
1.  **GUI (User Space):** Runs with standard user privileges, providing the dashboard and controls.
2.  **Privileged Helper (Root):** A background service with administrative privileges required for managing TUN interfaces and system routing tables.

These two processes must communicate to exchange commands (Start/Stop VPN), status updates (Connected/Disconnected), and real-time metrics (Bandwidth).

## Decision
For the initial macOS-focused implementation of the GUI, we will use **Unix Domain Sockets (UDS)** as the Inter-Process Communication (IPC) mechanism.

## Rationale
*   **Security:** Unix Domain Sockets are restricted to the local machine and do not expose any ports to the network. Access can be strictly controlled using standard filesystem permissions.
*   **Performance:** UDS has lower overhead than local TCP/UDP loops because it avoids the networking stack entirely.
*   **Simplicity:** Go's `net` package provides excellent support for `unix` network types, making implementation straightforward.
*   **Standardization:** UDS is the standard way for privileged daemons to communicate with user-space tools on Unix-like systems (macOS/Linux).

## Consequences
*   **Platform Specificity:** UDS is not natively available on Windows in the same way. When porting to Windows, the IPC layer will need an abstraction to use **Named Pipes**.
*   **Socket Management:** The helper tool must ensure the socket file (e.g., `/var/run/slopn.sock`) is created with the correct permissions (allowing the GUI user to read/write) and cleaned up on exit.
*   **Protocol:** We will define a simple JSON-RPC or custom binary protocol over the socket to facilitate structured communication.
