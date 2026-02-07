# ADR-Helper-Lifecycle: Always-On Background Service

## Status
Accepted

## Context
The SloPN client relies on a Privileged Helper Tool (macOS) or Windows Service to perform root-level operations like managing TUN interfaces and routing tables. We need to decide whether this service should only run while the GUI is open or persist in the background from system boot.

## Decision
We will implement the Privileged Helper Tool as an **Always-On background service**.

## Rationale
*   **Reliability:** An always-on service allows for robust auto-reconnection logic that persists even if the GUI application is closed or crashes.
*   **Security (Kill Switch):** To implement a reliable "Kill Switch" (blocking all non-VPN traffic if the tunnel drops), the networking logic must persist independently of the user-space GUI.
*   **User Experience:** Eliminates the startup delay associated with launching the privileged component and establishing IPC every time the GUI is opened.
*   **Standard Practice:** This aligns with the behavior of industry-standard VPN clients (e.g., Mullvad, Tailscale, WireGuard), which maintain a persistent "daemon" or "engine" process.

## Consequences
*   **Resource Usage:** A small amount of system memory and CPU will be consumed even when the VPN is not actively tunneling traffic. The helper must be designed to be extremely lightweight in its idle state.
*   **State Management:** The helper must maintain its own state (e.g., "should be connected") and synchronize this state with the GUI whenever the user opens the dashboard.
*   **System Integration:** Requires proper integration with system service managers (`launchd` on macOS, `Service Control Manager` on Windows) to handle automatic starts and restarts on failure.
