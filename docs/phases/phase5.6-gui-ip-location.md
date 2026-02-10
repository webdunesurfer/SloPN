# Phase 5.6: GUI IP & Geo-Location

**Goal:** Provide the user with immediate visual confirmation of their tunnel status by displaying their public IP, city, and country in the GUI dashboard.

## 1. Backend Implementation (Go)
*   **Service:** Use `http://ip-api.com/json` for lightweight, no-key-required geolocation.
*   **Method:** Add `GetPublicIPInfo()` to the `App` struct in `gui/app.go`.
*   **Data Structure:**
    ```go
    type IPInfo struct {
        Query   string `json:"query"`   // IP
        City    string `json:"city"`
        Country string `json:"country"`
        ISP     string `json:"isp"`
    }
    ```

## 2. Frontend Implementation (Svelte)
*   **Display:** Add a dedicated section in the `Status` card to show the IP and Location.
*   **Trigger:** 
    *   Fetch IP info on application startup.
    *   Automatically re-fetch immediately after the state changes to `connected`.
    *   Automatically re-fetch immediately after the state changes to `disconnected`.
*   **Visual Polish:** Use Emojis for country flags (e.g., 🇫🇷, 🇩🇪) based on the returned country data.

## 3. Success Criteria
*   [ ] User sees their original ISP IP before connecting.
*   [ ] User sees the VPN Server IP and its location immediately after connecting.
*   [ ] The information is updated without manual refresh.
*   [ ] Failure to reach the IP API does not crash the GUI.
