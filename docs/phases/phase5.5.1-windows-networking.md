# Phase 5.5.1: Windows Networking Layer

**Goal:** Enable the SloPN Helper to create TUN interfaces, configure IP addresses, and manage routing tables on Windows.

## 1. TUN Interface (WinTUN)

*   **Driver:** Use **WinTUN** (Layer 3 TUN driver). It is the standard for high-performance VPNs on Windows (used by WireGuard).
*   **Library:** Extend `pkg/tunutil` to support Windows.
    *   Investigate `water` library support for `wintun`. Note that `water` might default to the older `TAP-Windows6` driver.
    *   **Action:** If `water` is insufficient, switch `pkg/tunutil` to use `golang.zx2c4.com/wintun` or `golang.zx2c4.com/wireguard/tun` for Windows builds.

## 2. IP Configuration

*   **Mechanism:** Use `netsh` or Win32 APIs (IP Helper API) to assign the Virtual IP (VIP).
*   **Implementation:**
    *   Create `pkg/tunutil/tun_windows.go`.
    *   Implement `configureWindows(ifce, cfg)`.
    *   Command: `netsh interface ip set address "InterfaceName" static <IP> <Mask>`

## 3. Routing & DNS

*   **Routing Logic:**
    *   **Split Tunnel:** Add route for `10.100.0.0/24` via the TUN interface.
    *   **Full Tunnel:**
        *   Determine the metric of the physical interface.
        *   Add a specific route to the VPN Server Public IP via the physical gateway (to prevent routing loops).
        *   Add two routes `0.0.0.0/1` and `128.0.0.0/1` pointing to the TUN interface (standard override trick), or modify the default gateway metric.
*   **DNS:**
    *   Force the TUN interface DNS server to `10.100.0.1`.
    *   Command: `netsh interface ip set dns "InterfaceName" static 10.100.0.1`

## Deliverables
*   `pkg/tunutil` compiles and runs on Windows.
*   `cmd/tun-test` verifies interface creation and ping traffic.
