# GUI & Helper Connectivity: Lessons Learned

Our Phase 5.2 debugging session revealed several non-trivial obstacles related to macOS networking and inter-process communication.

## 1. Lessons Learned

### macOS Security & Sandboxing (IPC)
*   **Unix Sockets vs. TCP:** App Bundles (`.app`) on macOS are subject to strict sandboxing. Even with `777` permissions, connecting to a Unix Domain Socket owned by `root` in `/tmp` or the project directory was unreliable or blocked.
*   **Solution:** Switching to a local TCP socket (`127.0.0.1`) bypassed these filesystem-level sandbox restrictions and provided a more stable IPC bridge.

### UDP Binding Restrictions (`sendmsg`)
*   **The 0.0.0.0 Trap:** macOS prevents a `root` process from using `0.0.0.0` as a source address for outbound UDP packets in many contexts. Attempting to `Dial` via an unbound UDP socket resulted in the cryptic `sendmsg: can't assign requested address` error.
*   **Solution:** We must explicitly detect the active local interface IP (e.g., your Wi-Fi or Ethernet IP) and bind the UDP socket to that specific address before initiating the QUIC handshake.

### The "Ghost Process" Problem
*   **Persistence:** Because the Helper runs as `root` via `sudo`, it often persists even after the parent terminal is closed or the GUI is killed. These "zombie" helpers hold onto ports and sockets, causing "Connection Refused" or "Bind: Address already in use" for new versions.
*   **Solution:** Aggressive process cleanup (`killall`) and version tagging (`[V18]`) are essential during development to ensure we are testing the intended code.

### Routing Table Deadlocks
*   **Network Unreachable:** If a previous "Full Tunnel" attempt failed, it could leave the system with no valid default gateway, preventing the helper from even reaching the VPN server to start a new connection.
*   **Solution:** The helper must add a direct **host route** to the VPN server via the *original* gateway *before* attempting the connection.

## 2. Improvement Plan

### Helper Robustness
*   **Graceful Signal Handling:** Ensure `SIGTERM` and `SIGINT` always trigger the routing restoration logic.
*   **Privileged Helper Protocol:** Transition from raw TCP to a more secure IPC like **XPC** (macOS native) or authenticated local TCP to prevent other local users from hijacking the VPN.
*   **Interface Detection:** Automate the detection of the primary network interface to handle switching between Wi-Fi and Ethernet seamlessly.

### GUI Enhancements
*   **State Synchronization:** The GUI should more intelligently handle the "Helper Connecting" state so the user isn't tempted to click the button multiple times (preventing command spam).
*   **Auto-Update Logic:** Implement a more robust polling mechanism that can handle the helper restarting without losing the UI state.
*   **Log Streaming:** Instead of reading files, implement a log-streaming IPC command so the user can see the "Engine" logs directly in the Svelte dashboard.

### Packaging
*   **Helper Installation:** Move toward the `.pkg` installer (Phase 5.3) to place the helper in `/Library/PrivilegedHelperTools` properly, which will resolve most of the "run manually" permission issues.
