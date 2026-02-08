# SloPN Build and Release Guide

This document provides comprehensive instructions for building, packaging, and releasing SloPN both locally and via CI/CD.

## 1. Local Development Build

### Prerequisites
- **Go 1.25+**
- **Node.js 20+** & **npm**
- **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **macOS Dependencies**: Xcode Command Line Tools

### Building the Linux Server (Dockerized)
The recommended way to deploy the server is using the **One-Click Installer**:
```bash
curl -sSL https://raw.githubusercontent.com/webdunesurfer/SloPN/main/install-server.sh | bash
```

To build manually:
```bash
# 1. Compile Server
GOOS=linux GOARCH=amd64 go build -o bin/server_linux ./cmd/server/main.go

# 2. Start Infrastructure
docker compose up -d
```
*Note: Ensure `coredns.conf` is present in the root directory for the DNS container to start.*

### Building the macOS GUI (Wails)
From the `gui/` directory:
```bash
cd gui
# Install frontend dependencies
cd frontend && npm install && cd ..

# Build the universal .app bundle
export CGO_LDFLAGS="-framework Cocoa -framework UniformTypeIdentifiers"
wails build -platform darwin/universal
```

---

## 2. Local Packaging (macOS .pkg)

### 1. Prepare the Payload
```bash
mkdir -p packaging/payload/Applications packaging/payload/Library/PrivilegedHelperTools packaging/payload/Library/LaunchDaemons

# Copy binaries
cp -r gui/build/bin/SloPN.app packaging/payload/Applications/
cp bin/slopn-helper packaging/payload/Library/PrivilegedHelperTools/
cp packaging/com.webdunesurfer.slopn.helper.plist packaging/payload/Library/LaunchDaemons/

# IMPORTANT: Generate a dummy ipc.secret if it doesn't exist for packaging
# (The postinstall script will generate a real one on the target machine)
mkdir -p "/Library/Application Support/SloPN"
touch "/Library/Application Support/SloPN/ipc.secret"

find packaging/payload -name "._*" -delete
```

### 2. Build the Final Package
```bash
VERSION="0.3.8"
pkgbuild --root packaging/payload \
         --install-location / \
         --component-plist bin/SloPN-App.plist \
         --scripts packaging/scripts \
         --identifier com.webdunesurfer.slopn \
         --version "$VERSION" \
         bin/SloPN-Component.pkg

productbuild --package bin/SloPN-Component.pkg bin/SloPN-Installer.pkg
```

---

## 3. Automated Release (GitHub Actions)

1.  **Tag the commit**: `git tag v0.3.8`
2.  **Push the tag**: `git push origin v0.3.8`
3.  **CI Process**: The "Build and Package" workflow will automatically generate the Linux binary and macOS `.pkg` and create a GitHub Release draft.

---

## 4. Troubleshooting

### DNS Failures
If the `slopn-dns` container fails to start, check for port 53 conflicts on the host:
`sudo lsof -i :53`
The SloPN DNS service binds to the **Docker Bridge IP** to avoid these conflicts.

### Helper Authorization
If the GUI shows "Unauthorized," ensure `/Library/Application Support/SloPN/ipc.secret` exists and matches the secret the GUI is attempting to use.
