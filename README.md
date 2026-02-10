# SloPN: QUIC-based Layer 3 VPN

SloPN (Slow Private Network) is a modular, high-security VPN built with Go and QUIC. It utilizes RFC 9221 Datagrams to provide a robust tunneling experience that avoids the "TCP-over-TCP" meltdown common in older VPN protocols.

## 📖 Documentation Hub

For detailed information on various aspects of the project, please refer to the following documents:

### 🏛️ Architecture & Design
- **[System Architecture](docs/architecture.md)**: High-level overview of the protocol, data flow, and components.
- **[ADR Index](docs/adr/)**: Architectural Decision Records explaining the *why* behind key technical choices.

### 🛠️ Development & Operations
- **[Build and Release Guide](docs/build-and-release.md)**: Comprehensive instructions for local building, packaging, and CI/CD.
- **[Project Dependencies](docs/dependencies.md)**: Required drivers (WinTUN/TAP) and tools.
- **[GUI Dashboard](docs/gui-dashboard.md)**: Details on the Wails/Svelte-based user interface.
- **[Frontend Development](docs/frontend-development.md)**: Specifics on the Svelte frontend environment and tooling.
- **[Refactoring Plan](docs/refactoring-plan.md)**: Current roadmap for code improvements and technical debt.

### 📈 Project Roadmap
- **[Development Plan](docs/plan.md)**: Overall strategy and high-level milestones.
- **[Implementation Phases](docs/phases/)**: Step-by-step breakdown of the project's evolution.