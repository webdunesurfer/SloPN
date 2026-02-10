# Phase 5.5.3: Windows Packaging & Distribution

**Goal:** Create a professional installer for the Windows client.

## 1. Installer Logic

We will use **Inno Setup** (free, scriptable, reliable) or **WiX Toolset**. Inno Setup is recommended for simplicity with Go binaries.

### Installer Actions
1.  **Prerequisites:** Check if running as Admin.
2.  **Driver Installation (Critical):**
    *   **Option A (TAP-Windows - Current):**
        *   Bundle `tapinstall.exe` (from OpenVPN project) and the signed driver files (`OemVista.inf`, `tap0901.sys`, `tap0901.cat`).
        *   Execute: `tapinstall.exe install OemVista.inf tap0901` during setup.
        *   *Note:* This creates a persistent network adapter.
    *   **Option B (WinTUN - Recommended Alternative):**
        *   Bundle `wintun.dll` next to the executable.
        *   No separate driver installation required (DLL handles it).
        *   *Requires switching from 'water' to 'wireguard-go/tun' library.*
3.  **Files:**
    *   Install `SloPN.exe` (GUI).
    *   Install `slopn-helper.exe` (Service).
    *   Install `WebView2` runtime if missing (Wails dependency).
4.  **Service Registration:**
    *   Run `slopn-helper.exe -install` (if self-installing) or use `sc create`.
    *   Configure service to auto-start.
5.  **Firewall:**
    *   Add Windows Firewall rule to allow UDP traffic for `slopn-helper.exe`.

## 2. Uninstaller Logic
1.  **Remove Driver:** Run `tapinstall.exe remove tap0901` (if TAP was used).
2.  Stop the `SloPN Helper` service.
3.  Unregister/Delete the service.
4.  Remove files.
5.  Remove firewall rules.

## 3. Artifacts

*   `SloPN-Setup-x64.exe`

## Deliverables
*   Build script (`build-windows.bat` or `Taskfile` entry).
*   Inno Setup script (`packaging/windows/setup.iss`).
