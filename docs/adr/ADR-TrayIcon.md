# ADR-TrayIcon: Implementation Strategy for macOS Tray Icon

## Status
Proposed

## Context
The user requires a system tray icon (Menu Bar Extra) on macOS to provide quick access to VPN status and controls (Connect/Disconnect) without needing the main dashboard window open. Since the project uses **Wails v2**, which lacks native, stable tray support for macOS, we need to evaluate alternative implementation strategies.

## Options Evaluated

### 1. `getlantern/systray` (Go Library)
*   **Mechanism:** Uses a common cross-platform Go library for tray management.
*   **Pros:** Pure Go API.
*   **Cons:** **High risk of thread conflict.** On macOS, both Wails and `systray` attempt to manage the `NSApplication` main event loop. This frequently results in "deadlocks" or "duplicate symbol" errors during compilation.

### 2. Wails v3 Migration (Native API)
*   **Mechanism:** Port the GUI code to the upcoming Wails v3, which includes native Tray support.
*   **Pros:** Cleanest architecture, officially supported, multi-window capability.
*   **Cons:** Wails v3 is in Alpha/Beta. Requires significant refactoring of `main.go` and `app.go`.

### 3. Custom Objective-C Bridge (CGO)
*   **Mechanism:** Write a lightweight Objective-C bridge (`.m` file) to interface directly with the macOS `NSStatusItem` API, invoked from Go via CGO.
*   **Pros:** **Highest stability.** Minimal footprint. Avoids the "Main Thread" battle by only initializing specific native components needed for the icon.
*   **Cons:** Requires maintaining a small amount of non-Go code.

### 4. Separate Tray Process
*   **Mechanism:** Create a tiny standalone Go binary specifically for the tray icon, communicating with the same Helper IPC.
*   **Pros:** Decouples the tray from the GUI lifecycle; GUI crashes won't kill the tray.
*   **Cons:** Overhead of managing two separate UI binaries.

## Recommendation
We recommend **Option 3 (Objective-C Bridge)** for immediate stability within the current Wails v2 project.

## Decision
**Option 3: Custom Objective-C Bridge (CGO)**

We implemented a lightweight Objective-C wrapper (`tray_darwin.m`) that uses the native `NSStatusItem` and `NSMenu` APIs. This avoids conflicts with the Wails event loop by running on the main dispatch queue and using a unique delegate class.

### Key Implementation Details:
*   **Icon:** Uses native SF Symbols (`shield` and `shield.fill`).
*   **Coloring:** Uses `NSImageSymbolConfiguration` with hierarchical palette colors to provide a vivid green status indicator.
*   **Scaling:** Forces a `Large` scale to match standard macOS tray icon dimensions.
*   **Integration:** Initialized via Wails `OnStartup` hook.

