---
type: "query"
date: "2026-09-29T13:08:01.318840+00:00"
question: "Trace Windows installation, portable build, startup, executable replacement, configuration/data locations, and health checks. Design an atomic updater with backup and automatic rollback that preserves config, Vault, database, logs, and DNS state."
contributor: "graphify"
outcome: "useful"
---

# Q: Trace Windows installation, portable build, startup, executable replacement, configuration/data locations, and health checks. Design an atomic updater with backup and automatic rollback that preserves config, Vault, database, logs, and DNS state.

## Answer

Implemented scripts/update-windows.ps1 and portable integration. It validates/stages immutable files, exclusively locks updates, preserves configs/data, moves old binaries/web/desktop to timestamped rollback, requires exact /healthz service and VERSION identity, and automatically restores on failure. Portable package includes WPF desktop, updater, VERSION, and first-run guidance. Tests, package validation, and deployed identity health check passed.

## Outcome

- Signal: useful