# Phase 5.8: Windows CLI Client

**Goal:** Provide a lightweight, headless Command Line Interface (CLI) for Windows users, serving as a reliable alternative for older systems where the WebView2-based GUI might fail, and for automation/scripting purposes.

## 1. The Concept
We already have a robust, privileged "Engine" (`slopn-helper.exe`) running as a Windows Service. The current GUI communicates with it via local TCP (`127.0.0.1:54321`) using a Shared Secret.

We will build `slopn-cli.exe`—a simple Go binary that acts as a different "frontend" for the same Helper. It will reuse the exact same IPC protocol as the GUI.

## 2. Architecture
*   **Binary:** `cmd/cli/main.go` -> `bin/slopn-cli.exe`
*   **Communication:** Connects to `127.0.0.1:54321`.
*   **Authentication:** Reads `C:\ProgramData\SloPN\ipc.secret` to authorize itself with the Helper.
*   **Config:** Reads `C:\ProgramData\SloPN\config.json` (or accepts flags) for connection details.

## 3. Implementation Plan

### A. New CLI Package (`cmd/cli`)
Create a new main package that:
1.  **Reads the IPC Secret:** Just like the GUI, it must read the shared secret file.
2.  **Implements CLI Commands:**
    *   `slopn connect`: Reads config from disk and sends `CmdConnect`.
    *   `slopn connect --server <ip> --token <t> --obfs`: Connects with explicit flags.
    *   `slopn disconnect`: Sends `CmdDisconnect`.
    *   `slopn status`: Sends `CmdGetStatus` and prints JSON or human-readable output.
    *   `slopn logs`: Fetches and prints logs.

### B. Reuse Existing Logic
We can import `pkg/ipc` directly. We essentially just need a lightweight version of the `App` struct from `gui/app.go` that prints to Stdout instead of updating a Svelte store.

### C. Packaging
*   Update `build.yml` to compile `cmd/cli`.
*   Update `setup.iss` to include `slopn-cli.exe` in the installation directory.
*   Add the installation directory to the System `PATH` so users can just type `slopn` in PowerShell/CMD.

## 4. Proposed Commands
```powershell
slopn status
slopn connect
slopn disconnect
slopn logs
```

## 5. Success Criteria
*   [ ] `slopn status` shows correct state from the Helper.
*   [ ] `slopn connect` successfully triggers the tunnel.
*   [ ] The CLI works on Windows 10 (2016/2017 builds) where GUI might fail.
*   [ ] Installer adds `slopn` to PATH.
