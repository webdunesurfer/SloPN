# Phase 5.5.3: Windows GUI Adaptation

**Goal:** Ensure the Wails-based GUI works correctly on Windows, connecting to the privileged helper and respecting Windows filesystem conventions.

## 1. Filesystem & Path Adaptation
*   **Settings:** Store user settings (server list, preferences) in `%APPDATA%\SloPN\settings.json` instead of macOS Library paths.
*   **IPC Secret:** Read the IPC shared secret from `C:\ProgramData\SloPN\ipc.secret` to authenticate with the Helper.
*   **Implementation:** Use Go build tags for `paths_windows.go` and `paths_darwin.go`.

## 2. System Tray Integration
*   **Wails Runtime:** Use Wails built-in menu and tray capabilities where possible.
*   **Implementation:** Provide a `tray_windows.go` stub that correctly manages the application lifecycle on Windows (Minimize to tray, Close behavior).

## 3. Local Verification
*   **Build:** Compile the GUI using `wails build`.
*   **Test:** 
    1.  Ensure the Helper is running.
    2.  Launch the GUI.
    3.  Verify it loads the IPC secret.
    4.  Verify it can successfully send a "Connect" command and receive status updates.

## Deliverables
*   `SloPN.exe` (GUI) successfully controlling the Helper.
*   Build tags for all platform-specific GUI logic.
