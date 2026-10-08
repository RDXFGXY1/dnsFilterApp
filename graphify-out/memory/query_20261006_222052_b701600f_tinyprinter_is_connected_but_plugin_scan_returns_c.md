---
type: "query"
date: "2026-10-06T22:20:52.133324+00:00"
question: "TinyPrinter is connected but plugin scan returns context deadline exceeded; also open plugins in a popup"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Client", "Request"]
---

# Q: TinyPrinter is connected but plugin scan returns context deadline exceeded; also open plugins in a popup

## Answer

Expanded query to plugin, timeout, request, client, manifest, dialog, modal, workspace, action. Root cause: plugin host used a fixed 12-second client timeout while real Bluetooth scan took about 15.8 seconds. Added bounded per-action timeout_seconds (1-120), compatibility timeouts for already-installed manifests (scan 60s, calibration 120s, LaTeX 90s, default 45s), kept health at 5s, updated TinyPrinter manifest, and converted the inline workspace into an accessible modal with backdrop, Escape, Close, focus, and scroll locking. Tests passed, server restarted healthy, and direct scan completed with 2 devices in 15.8s.

## Outcome

- Signal: useful

## Source Nodes

- Client
- Request