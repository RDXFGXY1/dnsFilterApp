---
type: "query"
date: "2026-09-29T13:24:38.253909+00:00"
question: "Trace service startup and shutdown, Windows DNS routing setup/reset/status, DNS route watch behavior, adapter discovery, sleep/resume or network-change handling, autostart, and desktop health refresh. Identify reliability gaps for restart, resume, and adapter changes and the safest automated tests."
contributor: "graphify"
outcome: "useful"
source_nodes: ["Manager", "NewServer", "SetDNS", "ResetDNS", "Status"]
---

# Q: Trace service startup and shutdown, Windows DNS routing setup/reset/status, DNS route watch behavior, adapter discovery, sleep/resume or network-change handling, autostart, and desktop health refresh. Identify reliability gaps for restart, resume, and adapter changes and the safest automated tests.

## Answer

Expanded from graph vocabulary around DNS routing, watcher, startup, shutdown, Windows adapters, and tests. Implemented persisted opt-in routing intent, safe legacy migration only when loopback DNS is already active, serialized 30-second checks, automatic route repair, preserved explicit opt-out, graceful shutdown, CLI/API persistence, and simulated success/disabled/failure/migration tests. Full Go and WPF builds pass; deployed health is ready and live Wi-Fi DNS remains ::1/127.0.0.1. Actual restart, sleep/resume, and adapter switching remain manual release validation.

## Outcome

- Signal: useful

## Source Nodes

- Manager
- NewServer
- SetDNS
- ResetDNS
- Status