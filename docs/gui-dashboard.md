# SloPN GUI Dashboard

This is the user-facing dashboard for the SloPN VPN, built using the [Wails](https://wails.io/) framework with a **Svelte** frontend.

## 🏗️ Architecture

The GUI acts as a "remote control" for the **Privileged Helper** (`slopn-helper`).
- **Frontend**: Svelte + CSS (Material Design inspired).
- **Backend**: Go (Wails) acting as an IPC client.
- **IPC**: Communicates with the helper over local TCP port `54321`.

## 🚀 Getting Started

### Prerequisites
1.  **Wails CLI**: Install via `go install github.com/wailsapp/wails/v2/cmd/wails@latest`.
2.  **Node.js**: Required for frontend development.
3.  **SloPN Helper**: The helper must be running with root privileges for the GUI to function.

### Development Mode
Run the following in the `gui/` directory:
```bash
wails dev
```
This enables hot-reload for both Go and Svelte code.

### Building
To create a production-ready application bundle (`.app` on macOS):
```bash
wails build
```
The resulting binary will be in `build/bin/`.

## 🛠️ Features
- **Connection Management**: One-click connect/disconnect.
- **Real-time Stats**: Uptime, Bytes Sent, and Bytes Received.
- **Configuration**: In-app adjustment of Server Address, Auth Token, and Tunneling mode.
- **Version Tracking**: Displays versions for GUI, Engine (Helper), and Server.

## 📄 License
Licensed under the GNU General Public License v3.0.