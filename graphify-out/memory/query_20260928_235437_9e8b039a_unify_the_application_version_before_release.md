---
type: "query"
date: "2026-09-28T23:54:37.031247+00:00"
question: "Unify the application version before release"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", "Release Changelog", "Config"]
---

# Q: Unify the application version before release

## Answer

Expanded from graph vocabulary: release, server, config, dashboard. Added internal/version/VERSION as the canonical source with 2.6.0-beta.1, embedded it into Go, exposed it in the server banner and version subcommand, returned it from the dashboard system-info API, read it into WPF assembly metadata and displayed it in the desktop sidebar, and made portable packaging read the same file. Go version/API/config tests, server build/version command, and WPF Release build passed.

## Outcome

- Signal: useful

## Source Nodes

- Server
- Release Changelog
- Config