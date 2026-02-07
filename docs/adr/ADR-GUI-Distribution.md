# ADR-GUI-Distribution: Standard Installers (.pkg/.msi)

## Status
Accepted

## Context
The SloPN client requires a Privileged Helper Tool (macOS) or a Windows Service to perform administrative networking tasks. For a seamless "one-click" user experience, these background components must be correctly installed, registered with the OS, and granted the necessary permissions. We need to decide how to package and distribute the application to the end user.

## Decision
We will distribute the SloPN client using **Standard System Installers**:
*   **macOS:** `.pkg` (Installer Package)
*   **Windows:** `.msi` (Windows Installer) or a standard setup executable (e.g., via Inno Setup/NSIS).

## Rationale
*   **Privileged Component Setup:** Standard installers are designed to handle the installation of background daemons and services. This allows the Privileged Helper to be placed in the correct system directory (`/Library/PrivilegedHelperTools`) and registered during the installation process, rather than requiring the app to perform complex "self-elevation" on first run.
*   **User Trust & Experience:** Users expect professional VPN software to come with a standard installer that manages dependencies (like WinTUN on Windows) and provides an uninstaller.
*   **Security:** Using system installers allows for proper code signing and notarization, reducing "Unknown Developer" warnings and ensuring the integrity of the privileged components.
*   **Permissions:** Installers can pre-configure certain system permissions, making the first-run experience smoother for the user.

## Consequences
*   **Build Complexity:** Our CI/CD pipeline will need to be configured to generate these specific installer formats, which requires platform-specific tooling (e.g., `pkgbuild` on macOS, WiX or Inno Setup on Windows).
*   **Signing Requirements:** To avoid OS security blocks (like macOS Gatekeeper), the installers and the binaries within them must be signed with valid Developer certificates from Apple and Microsoft.
*   **Update Mechanism:** We will need to implement or integrate an auto-update system (e.g., Sparkle for macOS) that can handle downloading and running these installers for version updates.
