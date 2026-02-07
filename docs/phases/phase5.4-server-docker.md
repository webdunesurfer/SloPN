# Phase 5.4: Server Dockerization & Security

**Goal:** Package the SloPN Linux server into a Docker container to improve portability, security, and ease of deployment while maintaining full VPN functionality.

## Overview
Moving the server to Docker requires careful handling of Linux kernel networking features. The server relies on:
1.  **TUN Interface Creation:** Requires `/dev/net/tun` access.
2.  **IP Forwarding:** Requires `sysctl` modifications.
3.  **NAT/Masquerading:** Requires `iptables` and the `NET_ADMIN` capability.

## Tasks

### 1. Dockerfile Implementation
*   **Base Image:** Use a lightweight Alpine or Debian-slim image.
*   **Dependencies:** Install `iptables`, `iproute2`, and `ca-certificates`.
*   **Multi-Stage Build:** Compile the Go binary in a `golang` builder stage and copy it to the final lean image.

### 2. Networking Analysis & Container Strategy
*   **Privileged Mode vs. Capabilities:**
    *   To create TUN devices and manage `iptables`, the container needs `--cap-add=NET_ADMIN`.
    *   To modify `sysctl` (like `net.ipv4.ip_forward`), we either need `--privileged` or specific `sysctl` flags in `docker run`.
*   **Device Mapping:** Map `/dev/net/tun` from the host to the container.
*   **Environment Variables:** Allow configuring Port, Subnet, Token, and NAT via environment variables.

### 3. Docker Compose Setup
*   Create a `docker-compose.yml` for easy "one-click" deployment.
*   Configure **Restart Policy** to ensure the VPN stays up.
*   Explicitly handle the `NET_ADMIN` capability and `/dev/net/tun` device.

### 4. Security Hardening
*   **Non-Root User:** Investigate if the server can run as a non-root user within the container while still holding `NET_ADMIN` (using `setcap`).
*   **Minimal Surface:** Ensure only necessary UDP ports are exposed.

## Implementation Steps (Detailed)

### 1. The Entrypoint Script
A shell script will be used as the container entrypoint to:
1.  Enable IP forwarding if requested.
2.  Set up the initial `iptables` rules.
3.  Launch the `slopn-server` binary with the provided environment variables.

### 2. Host Integration
*   Ensure the host's kernel has the `tun` module loaded.
*   Analyze impact on host firewall (`ufw`/`firewalld`).

## Deliverables
*   `Dockerfile` in the project root.
*   `docker-compose.yml` for deployment.
*   Verified Dockerized VPN flow: Client connects to Dockerized Server -> Internet access via NAT works.
