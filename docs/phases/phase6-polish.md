# Phase 6: Cross-Platform & Polish

**Goal:** Refine the user experience, enhance security, and ensure stable performance across all supported platforms.

## 1. Security & IPC Enhancements

The current IPC mechanism (Local TCP + Shared Secret File) is secure against remote attackers but has weaknesses against local multi-user attacks due to file permissions (`0644`). We will evaluate and implement one of the following hardening strategies:

### Option A: Local mTLS (Cross-Platform Crypto)
Leverage Go's strong crypto library to authenticate the GUI via certificates instead of shared secrets.
*   **Mechanism:**
    *   Installer/First Run generates a Root CA.
    *   Helper gets a Server Cert signed by CA.
    *   GUI gets a Client Cert signed by CA, stored in the user's secure **Keyring**.
    *   Connection requires Mutual TLS handshake.
*   **Pros:** Uniform Go codebase for all OSs; strong "Possession" based security.
*   **Cons:** Complexity of managing local PKI/Certificate lifecycle.

### Option B: OS-Native Hardening (Identity Verification)
Use kernel-level features to verify the User Identity of the calling process.
*   **Windows:** Migrate TCP to **Named Pipes** (`\\.\pipe\slopn`) with ACLs restricting access to the specific User SID.
*   **macOS:** Migrate TCP to **Unix Domain Sockets** or **XPC**, using `SO_PEERCRED` or Entitlements to verify the caller's UID matches the allowed user.
*   **Pros:** Zero secret management; relies on OS Kernel trust.
*   **Cons:** Requires maintaining distinct, low-level platform code for each OS.

### Additional Security Tasks
*   **Firewall Kill Switch:** Implement a system-level "Kill Switch" (using `pf` on macOS and `WFP` on Windows) to block all non-VPN traffic if the tunnel unexpectedly drops.
*   **Brute-Force Protection:** Continue refining the Fail2Ban host integration.

## 2. GUI Refinement
*   **Log Streaming:** Implement a streaming IPC command to display the Engine's (Helper) real-time logs directly in the Svelte dashboard.
*   **State Machine Synchronization:** Robust handling of connecting/disconnecting states to prevent UI desync.
*   **System Tray Unification:** Refactor platform-specific tray implementations to use a unified library (e.g., `getlantern/systray`) for better maintainability.

## 3. Connectivity & Performance
*   **Interface Awareness:** Auto-reconnect when switching networks (Wi-Fi <-> Ethernet).
*   **Performance Tuning:** Path MTU (PMTU) discovery and batch packet processing.
*   **Auto-Reconnection:** Exponential backoff strategy for network drops.

## 4. Cross-Platform Support
*   ✅ **Windows Support:** Full integration with **TAP-Windows** and Windows Service (v0.5.x).
*   **Linux Desktop Support:** Integrate with `nmcli` or `systemd-resolved` for DNS management.