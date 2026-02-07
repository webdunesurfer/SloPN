# Phase 5.4: Windows Porting (Future)

**Goal:** Extend the Wails GUI and Helper architecture to support Windows users.

## Overview
Windows requires different mechanisms for privileged tasks, but the Wails UI and the core protocol logic remain the same.

## Tasks
*   **Windows Helper (Service):**
    *   Re-implement the helper tool as a **Windows Service**.
    *   Integrate with **WinTUN** for high-performance tunneling.
    *   Handle Windows routing table modifications via `netsh` or the IP Helper API.
*   **IPC Migration:**
    *   Implement **Named Pipes** as the IPC mechanism to replace Unix Sockets.
*   **Packaging:**
    *   Create an `.msi` or `.exe` installer (using WiX, NSIS, or Inno Setup).
    *   Handle WinTUN driver installation during setup.

## Deliverables
*   `SloPN-Installer.exe`.
*   Feature parity with the macOS client.
