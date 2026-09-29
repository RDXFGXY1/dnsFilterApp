---
type: "query"
date: "2026-09-28T20:53:43.695879+00:00"
question: "now tell me do tha app need somtihng to fix ?? befor the official release ?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", "Config", "SecurityConfig", "DNS Filter Configuration", "Release Changelog"]
---

# Q: now tell me do tha app need somtihng to fix ?? befor the official release ?

## Answer

Expanded from graph vocabulary: release, security, test, config, configuration, install, password, server. Audit result: automated Go tests, Go vet/build, JavaScript DNS bypass test, and WPF build passed. Official release is still blocked by unreproducible dirty/untracked source state, tracked runtime SQLite WAL/SHM data and user-specific recovery hash in configs/config.yaml, default credentials and placeholder JWT/unused HTTPS settings, version drift across server/dashboard/portable package, risky and incomplete Windows install/uninstall behavior including IPv6 disable and local CA installation, missing clean-install/upgrade/uninstall/elevated Windows and browser E2E testing, and an incomplete dependency vulnerability scan. Suitable for beta after cleanup, not official stable yet.

## Outcome

- Signal: useful

## Source Nodes

- Server
- Config
- SecurityConfig
- DNS Filter Configuration
- Release Changelog