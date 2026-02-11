# Phase 5.6: GUI IP & Geo-Location

**Goal:** Provide the user with immediate visual confirmation of their tunnel status by displaying their public IP, city, and country in the GUI dashboard.

## 1. Backend Implementation (Go)
*   **Service:** Use `http://ip-api.com/json` for lightweight, no-key-required geolocation.
*   **Method:** Add `GetPublicIPInfo()` to the `App` struct in `gui/app.go`.
*   **Hardening:** Uses `DisableKeepAlives: true` to force fresh TCP handshakes, ensuring requests respect new OS routes immediately after connection.
*   **Data Structure:**
    ```go
    type IPInfo struct {
        Query       string `json:"query"`
        City        string `json:"city"`
        Country     string `json:"country"`
        CountryCode string `json:"countryCode"`
        ISP         string `json:"isp"`
    }
    ```

## 2. Frontend Implementation (Svelte)
*   **Display:** A dedicated card below the Status section with a single-line layout.
*   **Trigger:** 
    *   Initial fetch on application startup.
    *   **Double-Tap Strategy:** On Connection/Disconnection, the UI shows "VERIFYING TUNNEL..." and performs two fetches: at 3 seconds and 7 seconds.
*   **Manual Refresh:** A dedicated refresh icon with "FETCHING IP INFO..." state.
*   **Visual Polish:** Uses reliable SVG flags via `flagsapi.com` to avoid Windows emoji limitations.

## 3. Success Criteria
*   [x] User sees their original ISP IP before connecting.
*   [x] User sees the VPN Server IP and its location reliably after routes settle (~7s).
*   [x] The information is updated automatically after state changes.
*   [x] Failure to reach the IP API does not crash the GUI.
*   [x] Windows-native flag display works correctly.