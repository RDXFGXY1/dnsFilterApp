---
type: "query"
date: "2026-09-29T19:14:33.894212+00:00"
question: "Which release paths can leak runtime secrets or local state?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["build-portable.ps1,Config,database.go,Release Changelog"]
---

# Q: Which release paths can leak runtime secrets or local state?

## Answer

Expanded via build certificate config database logs portable release script windows. The portable package is assembled by scripts/build-portable.ps1 from the Go server, sanitized config, web assets, updater/uninstaller, and optional WPF desktop. Runtime databases, logs, Vault data, certificates, private keys, debug symbols, developer paths, tokens, and unsafe archive paths must be excluded and audited after both folder assembly and ZIP creation.

## Outcome

- Signal: useful

## Source Nodes

- build-portable.ps1,Config,database.go,Release Changelog