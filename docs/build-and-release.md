# SloPN Build and Release Guide

This document provides comprehensive instructions for building, packaging, and releasing SloPN both locally and via CI/CD.

## 1. Local Development Build

To build the various components of SloPN manually for development or testing.

### Prerequisites
- **Go 1.25+** (The project currently uses features from recent Go versions)
- **Node.js 20+** & **npm** (For the GUI frontend)
- **Wails CLI**: Install via `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **macOS Dependencies**: Xcode Command Line Tools
- **Linux Dependencies**: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (for Wails development if running GUI on Linux)

### Building the Linux Server
From the project root:
```bash
mkdir -p bin
GOOS=linux GOARCH=amd64 go build -o bin/server_linux ./cmd/server/main.go
```

### Building the macOS Privileged Helper
From the project root:
```bash
mkdir -p bin
go build -o bin/slopn-helper ./cmd/helper/main.go
```

### Building the macOS GUI (Wails)
From the `gui/` directory:
```bash
cd gui
# Install frontend dependencies
cd frontend && npm install && cd ..

# Build the .app bundle
# Note: We use CGO_LDFLAGS to ensure proper framework linking on macOS
export CGO_LDFLAGS="-framework Cocoa -framework UniformTypeIdentifiers"
wails build -platform darwin/amd64 # or darwin/arm64, or darwin/universal
```
The resulting `SloPN.app` will be in `gui/build/bin/`.

---

## 2. Local Packaging (macOS .pkg)

To create the installer package locally on macOS.

### 1. Prepare the Payload
The installer needs to place files in specific system locations.
```bash
# Create structure
mkdir -p packaging/payload/Applications
mkdir -p packaging/payload/Library/PrivilegedHelperTools
mkdir -p packaging/payload/Library/LaunchDaemons

# Copy binaries
cp -r gui/build/bin/SloPN.app packaging/payload/Applications/
cp bin/slopn-helper packaging/payload/Library/PrivilegedHelperTools/
cp packaging/com.webdunesurfer.slopn.helper.plist packaging/payload/Library/LaunchDaemons/

# Clean up hidden macOS files that cause PKG errors
find packaging/payload -name "._*" -delete
```

### 2. Create Component Plist (First time or if structure changes)
```bash
pkgbuild --analyze --root packaging/payload bin/SloPN-App.plist
# Disable relocation so it always installs to /Applications
sed -i '' 's/<true\/>/<false\/>/g' bin/SloPN-App.plist
```

### 3. Build the Final Package
```bash
VERSION="0.1.7" # Replace with current version
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

The recommended way to release is by pushing a git tag.

1.  **Tag the commit**: `git tag v0.1.7`
2.  **Push the tag**: `git push origin v0.1.7`
3.  **CI Process**:
    - GitHub Actions will trigger.
    - It builds the Linux Server and macOS Installer.
    - It creates a **Draft Release** on GitHub.
4.  **Publish**: Go to the GitHub "Releases" page, edit the draft, and publish.

---

## 4. Troubleshooting Local Builds

### Wails build fails with "SloPN.app not found"
Ensure you are running `wails build` from the `gui/` directory and that the `wails.json` configuration is correct.

### Helper cannot bind to port 54321
The helper requires root privileges to run and bind to system ports. Ensure you run it with `sudo`.

### PKG installation fails
Check the macOS "Console.app" for errors during the installation process. Common issues involve the `postinstall` script failing to load the LaunchDaemon.