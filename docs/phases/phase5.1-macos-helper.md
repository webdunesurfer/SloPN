# Phase 5.1: The macOS Privileged Helper (The "Engine")

**Goal:** Move root-level networking logic into a standalone background daemon (`launchd`) to enable zero-password connections.

## Overview
The "Engine" is a Go-based background service that runs as root. It performs the privileged tasks that the GUI cannot.

## Tasks
*   **Code Extraction:**
    *   Extract TUN and Routing logic from `cmd/client` into a new `cmd/helper` package.
    *   Ensure the helper is standalone and can be compiled into a separate binary.
*   **Lifecycle Management:**
    *   Implement **Always-On** logic (ADR-Helper-Lifecycle).
    *   Handle system signals for graceful shutdown of tunnels.
*   **IPC Server:**
    *   Setup **Unix Domain Socket** server at `/var/run/slopn.sock` (ADR-GUI-IPC).
    *   Implement a JSON-RPC protocol for commands:
        *   `StartVPN(config)`: Create TUN, set routes, and start forwarding.
        *   `StopVPN()`: Tear down tunnel and restore routing.
        *   `GetStatus()`: Return connection state, uptime, and assigned VIP.
        *   `GetStats()`: Return real-time byte counts for bandwidth monitoring.

## Deliverables
*   `slopn-helper` binary.
*   Functional IPC server testable via `socat` or a small CLI tool.
