# Project Dependencies

This document outlines the system requirements, libraries, and drivers needed to build and run SloPN on various platforms.

## 🛠️ Build Environment (Cross-Platform)

To compile the project from source, you need:

*   **Go:** v1.25+ (Check `go.mod` for exact version).
*   **Node.js & npm:** Required for building the Svelte frontend (Wails).
*   **Wails CLI:** `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
*   **Git:** For version control.

---

##  macOS (Client)

### Runtime Requirements
*   **OS:** macOS 10.14 (Mojave) or newer (amd64/arm64).
*   **Privileges:** Administrator access is required to install the Helper tool and modify network settings.

### System Tools Used
The client utilizes standard macOS networking utilities:
*   `networksetup` (DNS configuration)
*   `route` (Routing table management)
*   `ifconfig` (Interface configuration)
*   `scutil` (DNS cache flushing)

---

## ⊞ Windows (Client)

### Runtime Requirements
*   **OS:** Windows 10 or Windows 11 (amd64).
*   **Drivers:** 
    *   **TAP-Windows Adapter V9 (`tap-windows6`):** Required by the `water` library.
        *   *Distribution:* The installer **must bundle** `tapinstall.exe` and the signed driver files (`.inf`, `.sys`, `.cat`) to create the adapter during setup.
*   **Privileges:** Administrator access (Run as Admin) is required to manage the Service and Network Adapter.

### System Tools Used
*   `netsh` (IP address assignment, DNS configuration)
*   `sc.exe` (Service Control Manager interaction)
*   `route.exe` (Routing table management)

### Packaging
*   **Inno Setup:** Recommended for creating the `.exe` installer.

---

## 🐧 Linux (Server)

### Runtime Requirements
*   **Kernel:** Linux Kernel 5.6+ (recommended for native WireGuard/UDP performance, though SloPN runs in user space).
*   **Container Runtime:** Docker & Docker Compose (Recommended deployment method).

### Native (Non-Docker) Requirements
If running the server binary directly:
*   **CAP_NET_ADMIN:** The process requires this capability.
*   **TUN Device:** `/dev/net/tun` must be accessible.
*   **IPRoute2:** `ip` command (for setting addresses on the interface).
*   **IPTables/NFTables:** For configuring NAT (Masquerading) if internet access is required for clients.
