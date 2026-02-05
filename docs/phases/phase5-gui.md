# Phase 5: Wails GUI Client

**Goal:** Wrap the macOS CLI client in a user-friendly GUI.

1.  **Wails Setup:** Initialize a Wails project.
2.  **Backend Integration:** Move Client logic into Wails `App` struct.
    *   Expose `Connect(config)` and `Disconnect()` methods to frontend.
    *   Stream logs/stats (upload/download rate) to frontend.
3.  **Frontend:**
    *   Simple React/Vue form for Server Address & Key.
    *   Connect/Disconnect toggle.
    *   Status indicator.
4.  **Deliverable:**
    *   `.app` bundle for macOS.
    *   **Test:** User clicks "Connect", successful connection to Phase 4 Server.
