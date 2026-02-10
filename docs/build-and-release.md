# SloPN Build and Release Guide

This document provides instructions for building, packaging, and releasing SloPN across all platforms.

## 1. Local Development Build

### Prerequisites
- **Go 1.25+**
- **Node.js 20+** & **npm**
- **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Windows**: Inno Setup 6+ (for packaging)
- **macOS**: Xcode Command Line Tools

### ⊞ Building for Windows
From the root directory:
```powershell
# 1. Build Helper
go build -o bin/slopn-helper.exe ./cmd/helper

# 2. Build GUI
cd gui
wails build -platform windows/amd64 -o SloPN.exe
cp build/bin/SloPN.exe ../bin/
```

###  Building for macOS
```bash
# 1. Build Helper
go build -o bin/slopn-helper ./cmd/helper

# 2. Build GUI
cd gui
export CGO_LDFLAGS="-framework Cocoa -framework UniformTypeIdentifiers"
wails build -platform darwin/universal
```

---

## 2. Packaging

### ⊞ Windows (.exe Installer)
We use **Inno Setup** to create the installer.
1. Ensure `bin/SloPN.exe` and `bin/slopn-helper.exe` are present.
2. Ensure TAP driver files are in `packaging/windows/driver/`.
3. Run the compiler:
```powershell
iscc packaging/windows/setup.iss
```
The installer will be generated at `bin/SloPN-Setup.exe`.

###  macOS (.pkg Installer)
1. Prepare the payload in `packaging/payload/`.
2. Run `pkgbuild`:
```bash
pkgbuild --root packaging/payload \
         --install-location / \
         --scripts packaging/scripts \
         --identifier com.webdunesurfer.slopn \
         --version "0.5.6" \
         bin/SloPN-Installer.pkg
```

---

## 3. Automated Release (GitHub Actions)

The project uses GitHub Actions to automate builds on every tag:
1.  **Tag the commit**: `git tag v0.5.6`
2.  **Push the tag**: `git push origin v0.5.6`
3.  **Artifacts**: CI will generate:
    - Linux server binary.
    - macOS `.pkg` installer.
    - Windows `.exe` installer.
    - Windows standalone binaries.

---

## 4. Troubleshooting

### Windows: "Failed to find tap device"
Ensure the **TAP-Windows Adapter V9** is installed. The installer handles this, but for manual builds, you may need to run `tapinstall.exe install OemVista.inf tap0901`.

### Helper IPC Failures
Check if `ipc.secret` exists in the protected system directory:
- **Windows:** `C:\ProgramData\SloPN\ipc.secret`
- **macOS:** `/Library/Application Support/SloPN/ipc.secret`