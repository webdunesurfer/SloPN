# ADR-GUI-IPC: Communication between GUI and Privileged Helper

## Status
Amended (Replaced UDS with local TCP)

## Context
To provide a "one-click" VPN experience on macOS without requiring a `sudo` password for every connection, the SloPN client architecture is split into two components:
1.  **GUI (User Space):** Runs with standard user privileges, providing the dashboard and controls.
2.  **Privileged Helper (Root):** A background service with administrative privileges required for managing TUN interfaces and system routing tables.

These two processes must communicate to exchange commands (Start/Stop VPN), status updates (Connected/Disconnected), and real-time metrics (Bandwidth).

## Decision
We will use **Local TCP Sockets (`127.0.0.1`)** on port **54321** as the Inter-Process Communication (IPC) mechanism. 

Starting from **v0.2.2**, all communication over this socket must be authenticated using a **Shared Secret** string.

## Rationale
*   **Sandbox Compatibility:** During development, it was discovered that macOS App Bundles (`.app`) are subject to strict sandboxing that prevents reliable access to Unix Domain Sockets (UDS) created by root in `/var/run` or `/tmp`. Local TCP bypasses these filesystem permission hurdles while remaining internal to the machine.
*   **Wails Integration:** Wails-based applications interact more predictably with network-based IPC than filesystem-based sockets when packaged as a macOS application.
*   **Security (Shared Secret):** Since TCP is visible to other local processes, the shared secret ensures that only authorized clients (like the SloPN GUI) can control the privileged helper. The secret is generated during installation and stored in a root-protected directory.

## Consequences
*   **Port Collision:** The helper must be able to handle cases where port 54321 is already in use.
*   **Security:** While limited to `127.0.0.1`, a TCP port is technically "visible" to other local processes. We mitigate this by ensuring the IPC protocol is strictly internal and will consider adding a "Local Shared Secret" authentication in Phase 6.
*   **Protocol:** We use a simple JSON protocol to facilitate structured communication.

