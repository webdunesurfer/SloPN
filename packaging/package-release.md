# SloPN Release Guide

This document outlines the process for building and releasing a new version of SloPN.

## 1. Local Build Process (Development & Testing)

To build the macOS Installer and Linux Server manually on your machine:

### Prerequisites
- Go 1.21+
- Node.js & NPM
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Build Steps
1.  **Update Version Numbers**:
    Ensure the following files reflect the new version (e.g., `0.1.4`):
    - `cmd/helper/main.go` (`HelperVersion`)
    - `cmd/server/main.go` (`ServerVersion`)
    - `gui/app.go` (`GUIVersion`)
    - `gui/frontend/src/App.svelte` (`guiVersion`)

2.  **Run the Build Script**:
    You can use the following combined command from the project root:
    ```bash
    # 1. Build Engine and Server
    go build -o bin/slopn-helper ./cmd/helper/main.go
    GOOS=linux GOARCH=amd64 go build -o bin/server_linux ./cmd/server/main.go

    # 2. Build GUI Dashboard
    cd gui/frontend && npm install && npm run build && cd ..
    CGO_LDFLAGS="-framework Cocoa -framework UniformTypeIdentifiers" go build -tags production -o build/bin/SloPN .
    
    # 3. Assemble macOS App Bundle
    cp build/bin/SloPN build/bin/SloPN.app/Contents/MacOS/gui
    
    # 4. Create Installer Payload
    cd ..
    mkdir -p packaging/payload/Applications packaging/payload/Library/PrivilegedHelperTools packaging/payload/Library/LaunchDaemons
    cp -r gui/build/bin/SloPN.app packaging/payload/Applications/
    cp bin/slopn-helper packaging/payload/Library/PrivilegedHelperTools/
    cp packaging/com.webdunesurfer.slopn.helper.plist packaging/payload/Library/LaunchDaemons/
    find packaging/payload -name "._*" -delete

    # 5. Generate final PKG
    pkgbuild --root packaging/payload --install-location / --component-plist bin/SloPN-App.plist --scripts packaging/scripts --identifier com.webdunesurfer.slopn --version 0.1.4 bin/SloPN-Component.pkg
    productbuild --package bin/SloPN-Component.pkg bin/SloPN-Installer.pkg
    ```

## 2. Automated Release via GitHub Actions (Recommended)

The project is configured with a CI pipeline that automates the build and release process.

### Steps to Release
1.  **Commit and Push** all changes to the `main` branch.
2.  **Tag the Release**:
    ```bash
    git tag v0.1.4
    git push origin v0.1.4
    ```
3.  **Monitor the Action**:
    Go to the **Actions** tab on your GitHub repository. The "Build and Package" workflow will trigger.
4.  **Finalize the Release**:
    Once the build is complete, go to **Releases**. A new **Draft** will be created with:
    - `SloPN-Installer.pkg` (macOS)
    - `server_linux` (Linux Server)
    - Automatically generated release notes.
5.  Click **Edit** on the draft, review the notes, and click **Publish Release**.

## 3. Deployment to Testing Server

To deploy the new server binary to your remote testing machine:

```bash
# Upload
scp bin/server_linux vkh@your-server-ip:~/SloPN/server.new

# Restart (SSH into server)
sudo mv ~/SloPN/server.new ~/SloPN/server
sudo pkill -f /home/vkh/SloPN/server
sudo nohup /home/vkh/SloPN/server -nat > /home/vkh/SloPN/server.log 2>&1 &
```

## 4. Troubleshooting
- **Permission Denied**: Ensure the `postinstall` and `preinstall` scripts in `packaging/scripts/` have `chmod +x` permissions.
- **Old Branding**: If the app name doesn't update, the `preinstall` script handles removal of the old bundle, but a manual `sudo rm -rf /Applications/SloPN.app` may be required if permissions are stuck.
