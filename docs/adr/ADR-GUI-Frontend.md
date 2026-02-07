# ADR-GUI-Frontend: Svelte for Wails Dashboard

## Status
Accepted

## Context
The SloPN client requires a responsive, lightweight, and modern graphical user interface (GUI). We have chosen **Wails** as the cross-platform framework to bridge our Go backend logic with a web-based frontend. We need to select a specific frontend framework/library to build the dashboard UI.

## Decision
We will use **Svelte** as the primary frontend framework for the SloPN desktop application.

## Rationale
*   **Performance:** Svelte compiles the UI into highly optimized vanilla JavaScript at build time, rather than using a Virtual DOM at runtime. This results in faster startup times and lower memory usage, which is critical for a background utility like a VPN client.
*   **Small Footprint:** Svelte's minimal runtime overhead aligns with our goal of keeping the SloPN client lightweight.
*   **Developer Experience:** Svelte's component-based architecture is intuitive and reduces boilerplate code, allowing for rapid development of the connection dashboard and settings panels.
*   **Wails Integration:** Wails has excellent first-class templates and support for Svelte, making the setup and Go-to-JS binding process seamless.

## Consequences
*   **Learning Curve:** While Svelte is known for its simplicity, developers familiar only with React or Vue will need to learn Svelte's specific syntax for reactivity and state management.
*   **Ecosystem:** While Svelte's ecosystem is growing rapidly, it is smaller than React's. However, for a VPN dashboard, the available libraries (Tailwind CSS, etc.) are more than sufficient.
*   **Bundle Size:** The resulting `.app` or `.exe` will benefit from a smaller internal web bundle compared to React-based alternatives.
