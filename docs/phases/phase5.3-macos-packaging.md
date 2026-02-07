# Phase 5.3: Installation & Packaging (.pkg)

**Goal:** Create a standard macOS installer that handles the complex setup of privileged components automatically.

## Overview
A VPN app cannot be "drag-and-dropped" if it requires a root helper. A `.pkg` installer is required to place files in system directories and register the daemon.

## Tasks
*   **Directory Structure:**
    *   `SloPN.app` -> `/Applications/`
    *   `slopn-helper` -> `/Library/PrivilegedHelperTools/`
    *   `com.webdunesurfer.slopn.helper.plist` -> `/Library/LaunchDaemons/`
*   **Launchd Configuration:**
    *   Create the `.plist` file to ensure the helper starts at boot and restarts on crash.
*   **Installer Creation:**
    *   Use `pkgbuild` to create component packages.
    *   Use `productbuild` to create the final distribution package.
    *   Implement "Post-Install" scripts to load the `launchd` daemon immediately.
*   **Security:**
    *   Setup code signing for the app and the helper.
    *   (Optional) Notarize the package for distribution outside the Mac App Store.

## Deliverables
*   `SloPN-Installer.pkg`.
*   Verified installation flow: User runs PKG -> App and Helper are installed -> User opens App -> VPN connects with one click.
