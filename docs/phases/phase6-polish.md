# Phase 6: Cross-Platform & Polish

**Goal:** Refine the user experience, enhance security, and ensure stable performance across all supported platforms.

## 1. Security & IPC Enhancements
*   **Secure IPC:** Transition from local TCP to more secure, platform-native IPC mechanisms:
    *   **macOS:** XPC Services for secure GUI-to-Helper communication.
    *   **Windows:** Authenticated Named Pipes.
*   **Brute-Force Protection (Fail2Ban):** Implement a security layer to block attackers attempting to guess the Auth Token.
    *   **Server Logging:** Ensure the Go server logs failed authentication attempts with the remote IP in a structured format (e.g., `[AUTH_FAILURE] <remote_ip>`).
    *   **Host Integration:** Configure `fail2ban` on the host machine to monitor Docker container logs using the `docker` log driver or a shared log file, automatically banning IPs after multiple failed attempts via `iptables`.
*   **Firewall Kill Switch:** Implement a system-level "Kill Switch" (using `pf` on macOS and `WFP` on Windows) to block all non-VPN traffic if the tunnel unexpectedly drops.

## 2. GUI Refinement
*   **Log Streaming:** Implement a streaming IPC command to display the Engine's (Helper) real-time logs directly in the Svelte dashboard for easier troubleshooting.
*   **State Machine Synchronization:** Improve the GUI state machine to handle connecting/disconnecting states more robustly, preventing duplicate command execution and providing better visual feedback.
*   **Advanced Settings:** Add options for custom DNS, split tunneling, and protocol selection.

## 3. Connectivity & Performance
*   **Interface Awareness:** Automatically detect and respond to network interface changes (e.g., switching from Wi-Fi to Ethernet) by re-binding the tunnel source address.
*   **Performance Tuning:** 
    *   Implement Path MTU (PMTU) discovery to optimize packet sizes.
    *   Implement batch packet reading/writing to reduce CPU overhead.
*   **Auto-Reconnection:** Add an exponential backoff strategy for automatic reconnection after a network drop.

## 4. Cross-Platform Support
*   **Windows Support:** Full integration with **WinTun** and implementation of the Windows Service version of the helper.
*   **Linux Desktop Support:** Integrate with `nmcli` or `systemd-resolved` for DNS management.