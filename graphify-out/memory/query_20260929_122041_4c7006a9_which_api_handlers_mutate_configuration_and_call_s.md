---
type: "query"
date: "2026-09-29T12:20:41.902736+00:00"
question: "Which API handlers mutate configuration and call SaveDefault without handling errors, and what runtime state must be rolled back?"
contributor: "graphify"
outcome: "useful"
source_nodes: [".SaveDefault()", "Config", "Engine", "DNSCache"]
---

# Q: Which API handlers mutate configuration and call SaveDefault without handling errors, and what runtime state must be rolled back?

## Answer

The persistence fix made each Config retain its own loaded path, changed writes to same-directory temporary-file replacement, added ReplaceWith for Vault restore, serialized API configuration mutations, and added rollback for module toggles, block action, password changes, blocklist source add/toggle/remove, and auto-update interval changes. Failed saves now return config_save_failed rather than false success. Tests force save failures and verify state remains unchanged; full Go tests and Windows deployment passed.

## Outcome

- Signal: useful

## Source Nodes

- .SaveDefault()
- Config
- Engine
- DNSCache