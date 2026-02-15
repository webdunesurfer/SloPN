# Implementation Plan: Custom QUIC-based VPN (Hub-and-Spoke)

## Overview
**Project Name:** SloPN
**Architecture:** Hub-and-Spoke
**Stack:** Go, `quic-go` (RFC 9221 Datagrams), `water` (TUN), Wails.

This plan is divided into distinct phases. Each phase builds upon the previous one and results in a deployable, testable artifact.

## Roadmap Phases

1.  **[Phase 1: The Transport Layer](phases/phase1-transport.md)**
2.  **[Phase 2: Point-to-Point Tunnel](phases/phase2-tunnel.md)**
3.  **[Phase 3: Hub-and-Spoke Routing](phases/phase3-routing.md)**
4.  **[Phase 3.1: Reliable ICMP & Routing](phases/phase3.1-icmp-routing.md)**
5.  **[Phase 3.2: Multi-Client Routing](phases/phase3.2-multi-client.md)**
6.  **[Phase 4: Authentication & Internet Access](phases/phase4-auth-nat.md)**
7.  **[Phase 5: GUI Development](phases/phase5-gui.md)**
    *   *Includes macOS Helper, Windows Port, IP Location, and Packaging (5.1 - 5.8)*
8.  **[Phase 5.7: Basic Protocol Obfuscation](phases/phase5.7-protocol-obfuscation.md)** (Current XOR)
9.  **[Phase 6: Reality-Style Stealth Transport](phases/phase6-reality-transport.md)** (Next Gen)
10. **[Phase 7: Final Polish & Optimization](phases/phase7-polish.md)**
