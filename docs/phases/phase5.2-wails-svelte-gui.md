# Phase 5.2: Wails Dashboard with Svelte

**Goal:** Build the user-facing dashboard using the Wails framework to provide a modern, easy-to-use interface.

## Overview
The GUI acts as the "remote control" for the Privileged Helper. It runs in user space and communicates via Unix Domain Sockets.

## Tasks
*   **Wails Initialization:**
    *   Initialize Wails project with the **Svelte** template (ADR-GUI-Frontend).
    *   Configure the application to run as a **Menu Bar Extra** (Tray App).
*   **Backend (Go):**
    *   Implement the IPC client to connect to `/var/run/slopn.sock`.
    *   Bind Go methods to the frontend: `Connect()`, `Disconnect()`, `SaveConfig()`.
    *   Implement a background poller to fetch stats from the helper and push them to the frontend.
*   **Frontend (Svelte + Tailwind):**
    *   **Connection View:** Large toggle button, status text, and assigned VIP.
    *   **Settings View:** Form for Server Address and Token.
    *   **Stats View:** Real-time bandwidth graphs (Upload/Download).
    *   **Logs View:** Tail the logs from the helper tool for debugging.

## Deliverables
*   Functional `SloPN.app` (requires helper to be running).
*   Responsive and reactive UI that updates as the connection state changes.
