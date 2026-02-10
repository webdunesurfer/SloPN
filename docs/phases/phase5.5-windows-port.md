# Phase 5.5: Windows Porting

**Goal:** Extend the SloPN client to support Windows 10/11.

## 🌍 Strategic Requirement: Multi-Platform Client Solution

**Fact:** We must deliver a multi-platform client solution.
*   The architecture must allow the **GUI** (Wails) and **Helper** (privileged engine) to operate seamlessly on both macOS and Windows.
*   **Helper Strategy:** We must prioritize a **unified helper codebase** where possible, utilizing Go's build tags (`_windows.go`, `_darwin.go`) to handle platform specific logic (TUN creation, routing, IPC). A completely separate "Windows Helper" binary should only be considered if the divergence in logic becomes unmanageable. The goal is `cmd/helper` building for both OS targets.

## Sub-Phases

To manage the complexity of the Windows port, this phase is split into three distinct work packages:

*   **[Phase 5.5.1: Networking Layer](phase5.5.1-windows-networking.md)**
    *   Implementing `WinTUN` support.
    *   Adapting IP configuration and Routing logic (`netsh`, IP Helper API).
    *   DNS management on Windows.

*   **[Phase 5.5.2: Service & IPC](phase5.5.2-windows-service-ipc.md)**
    *   Running the Helper as a **Windows Service**.
    *   Migrating IPC from Unix Sockets to **Named Pipes**.
    *   Secure storage integration (Windows Credential Manager).

*   **[Phase 5.5.3: Packaging & Distribution](phase5.5.3-windows-packaging.md)**
    *   Creating the `.exe` / `.msi` installer.
    *   Bundling drivers (`wintun.dll`).
    *   Firewall and Service registration hooks.

## Success Criteria (Overall)
*   [ ] Single `cmd/helper` codebase compiles for both macOS and Windows.
*   [ ] Windows Client can connect to the VPN server.
*   [ ] Feature parity with macOS (Split/Full Tunnel, DNS protection).
