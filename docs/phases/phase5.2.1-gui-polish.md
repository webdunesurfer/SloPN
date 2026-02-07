# Phase 5.2.1: GUI Polish & Log Streaming

**Goal:** Complete the missing features from Phase 5.2 and improve the overall user experience and observability.

## Overview
This sub-phase focuses on transition from a standard window app to a Menu Bar app, adding log visibility, and improving UI responsiveness.

## Tasks
*   **Menu Bar Integration (macOS):**
    *   ⚠️ **Note:** Built-in Tray/Menu Bar support is limited in Wails v2.11.0. Reverted to fixed-size standard window (800x650) for stability.
*   **Log Streaming System:**
    *   ✅ **Helper:** Implement an IPC command `CmdGetLogs` to return the last N lines of `helper.log`.
    *   ✅ **GUI (Go):** Add a method to fetch these logs and push them via Wails events.
    *   ✅ **Frontend (Svelte):** Add a "Logs" section at the bottom of the dashboard with an auto-scrolling text area.
*   **UI Polish & Safety:**
    *   ✅ **Button Debouncing:** Disable the "CONNECT" button while the state is `connecting`.
    *   ✅ **Connection Timeout:** If connection takes longer than 15 seconds, show a "Timeout" error and reset state.
*   **Visual Bandwidth Graphs:**
    *   ✅ **Implementation:** Added SVG Sparklines to visualize `bytes_sent` and `bytes_recv` delta over time.

## Deliverables
*   SloPN GUI running as a Menu Bar app.
*   Real-time log visibility within the dashboard.
*   Visual bandwidth tracking.
