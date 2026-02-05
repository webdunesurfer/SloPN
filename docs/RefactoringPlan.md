# Refactoring & Improvement Plan

Based on the Phase 4 implementation, the following areas are identified for improvement to move toward a production-ready VPN.

## 1. Security Enhancements
- **mTLS Implementation:** Replace self-signed certificates with a proper Certificate Authority (CA) system where both client and server verify each other's certificates.
- **Dynamic Token Exchange:** Instead of static tokens in `config.json`, implement a challenge-response mechanism.
- **Firewall Hardening:** Add server-side `iptables` rules to strictly isolate clients from each other if needed (Disable "Spoke-to-Spoke" by default).

## 2. Robustness & Error Handling
- **Keep-Alives/Heartbeats:** While QUIC has built-in keep-alives, explicit application-level health checks would improve reconnection logic.
- **Auto-Reconnection:** The client currently exits on connection loss. Implement an exponential backoff retry strategy.
- **MTU Path Discovery:** Currently, MTU is hardcoded to 1280. Implement PMTU (Path MTU Discovery) to optimize packet sizes for different networks.

## 3. Architecture & Performance
- **Zero-Copy Data Path:** Reduce the number of `[]byte` allocations and copies in the `TUN <-> QUIC` loop using a buffer pool (`sync.Pool`).
- **Configuration Management:** Move beyond simple JSON files to support environment variables and a more robust CLI interface (e.g., using `cobra` or `urfave/cli`).
- **Multi-Platform Server Support:** Extend the server configuration logic to support macOS and Windows as VPN gateways (currently Linux-optimized).

## 4. Observability
- **Metrics:** Export Prometheus metrics for bandwidth usage, active sessions, and handshake latencies.
- **Structured Logging:** Replace `fmt.Printf` and `log.Fatalf` with a structured logger like `zap` or `slog` for better log analysis.

## 5. Deployment
- **Systemd Integration:** Create a systemd unit file for the Linux server to ensure it starts on boot.
- **Dockerization:** Containerize the server with all necessary `iptables` and `sysctl` configurations bundled.
