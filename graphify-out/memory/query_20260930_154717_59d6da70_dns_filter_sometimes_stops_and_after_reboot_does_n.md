---
type: "query"
date: "2026-09-30T15:47:17.038822+00:00"
question: "DNS Filter sometimes stops and after reboot does not work until the GUI opens"
contributor: "graphify"
outcome: "useful"
source_nodes: ["autostart_windows.go", "Server", "build-portable.ps1"]
---

# Q: DNS Filter sometimes stops and after reboot does not work until the GUI opens

## Answer

Expanded through graph vocabulary: autostart, dns, install, process, restart, running, server, service, start, windows. The runtime had no independent boot service and the GUI fallback launched the server. Logs also showed fatal API/DNS listener exits and bind conflicts from duplicate instances. Implemented a SYSTEM Scheduled Task with startup and logon triggers, IgnoreNew, battery-safe unlimited runtime, and one-minute restart-on-failure; the installer now pauses the task during updates, refreshes it automatically, launches the service through the task, and handles rollback without duplicate restarts. The main Go process now coordinates listener failures, treats http.ErrServerClosed as normal, shuts down cleanly, and returns non-zero only for genuine listener failure. The desktop now discovers installed dns-filter.exe. Registered the repaired task, replaced the old process, verified health for 2.6.0-beta.1, ran Go/Windows tests, desktop build, release audit, and SHA verification.

## Outcome

- Signal: useful

## Source Nodes

- autostart_windows.go
- Server
- build-portable.ps1