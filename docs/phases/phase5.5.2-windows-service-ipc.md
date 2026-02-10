# Phase 5.5.2: Windows Service & IPC

**Goal:** Adapt the Helper process to run as a Windows Service and communicate securely with the GUI.

## 1. Windows Service Wrapper

The `cmd/helper` must be able to run as a background service managed by the Service Control Manager (SCM).

*   **Library:** `golang.org/x/sys/windows/svc`.
*   **Implementation:**
    *   Refactor `cmd/helper/main.go` to separate the "app logic" from the "entry point".
    *   Create `cmd/helper/service_windows.go` implementing `svc.Handler`.
    *   Handle SCM signals: `svc.Stop`, `svc.Shutdown`.

## 2. IPC Adaptation (Unified Local TCP)

*   **Decision:** We will continue using **Local TCP (`127.0.0.1:54321`)** for Windows, maintaining parity with macOS.
*   **Security:** The existing **Shared Secret** mechanism will be used to authorize the GUI.
*   **Refactoring:**
    *   Ensure the helper's TCP listener is robust on Windows.
    *   The shared secret file will be stored in a protected system directory (e.g., `C:\ProgramData\SloPN\ipc.secret`).
    *   The GUI will read this secret to authenticate its requests.

## 3. Secure Storage

*   **Component:** `gui` and `helper`.
*   **Mechanism:** Windows Credential Manager.
*   **Library:** `github.com/zalando/go-keyring` (already in use) supports Windows Credential Manager transparently.
*   **Verification:** Ensure keys are stored/retrieved correctly.

## Deliverables
*   `slopn-helper.exe` can be registered and started via `sc start`.
*   GUI can send "Connect" command to the running service via Local TCP.
