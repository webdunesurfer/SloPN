# Phase 5: GUI & Native Integration

**Goal:** Transition SloPN from a CLI tool to a professional desktop application for macOS and Windows.

## Sub-Phases
This phase is divided into modular steps to ensure a stable and secure rollout:

1.  **[Phase 5.1: macOS Privileged Helper](phase5.1-macos-helper.md)** - Building the background engine.
2.  **[Phase 5.2: Wails Dashboard with Svelte](phase5.2-wails-svelte-gui.md)** - Building the user interface.
3.  **[Phase 5.3: macOS Packaging (.pkg)](phase5.3-macos-packaging.md)** - Creating the installer.
4.  **[Phase 5.4: Server Dockerization](phase5.4-server-docker.md)** - Containerizing the server for secure deployment.
5.  **[Phase 5.5: Windows Porting](phase5.5-windows-port.md)** - Extending to Windows.

## Architectural Decisions
All implementations must follow the established ADRs:
*   [ADR-GUI-IPC: Unix Domain Sockets](../adr/ADR-GUI-IPC.md)
*   [ADR-GUI-Frontend: Svelte Framework](../adr/ADR-GUI-Frontend.md)
*   [ADR-GUI-Distribution: Standard Installers](../adr/ADR-GUI-Distribution.md)
*   [ADR-Helper-Lifecycle: Always-On Service](../adr/ADR-Helper-Lifecycle.md)
