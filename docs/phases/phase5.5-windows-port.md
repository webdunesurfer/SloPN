# Phase 5.5: Windows Porting

**Goal:** Extend the SloPN client to support Windows 10/11.

## 🌍 Strategic Requirement: Multi-Platform Client Solution

**Fact:** We must deliver a multi-platform client solution.
*   The architecture must allow the **GUI** (Wails) and **Helper** (privileged engine) to operate seamlessly on both macOS and Windows.
*   **Helper Strategy:** We must prioritize a **unified helper codebase** where possible, utilizing Go's build tags (`_windows.go`, `_darwin.go`) to handle platform specific logic (TUN creation, routing, IPC).

## Sub-Phases

*   **[Phase 5.5.1: Networking Layer](phase5.5.1-windows-networking.md)**
    *   Implementing `WinTUN` support.
    *   Adapting IP configuration and Routing logic.

*   **[Phase 5.5.2: Service & IPC](phase5.5.2-windows-service-ipc.md)**
    *   Running the Helper as a **Windows Service**.
    *   Unified **Local TCP IPC** with shared secret.

*   **[Phase 5.5.3: GUI Adaptation](phase5.5.3-windows-gui.md)**
    *   Platform-specific path handling (`%APPDATA%`).
    *   Windows Tray integration.
    *   Local build and verification.

*   **[Phase 5.5.4: Packaging & Distribution](phase5.5.4-windows-packaging.md)**
    *   Creating the `.exe` installer.
    *   Bundling drivers and service registration.

## Success Criteria (Overall)
*   [ ] Single `cmd/helper` codebase compiles for both macOS and Windows.
*   [ ] Windows Client can connect to the VPN server.
*   [ ] Feature parity with macOS (Split/Full Tunnel, DNS protection).