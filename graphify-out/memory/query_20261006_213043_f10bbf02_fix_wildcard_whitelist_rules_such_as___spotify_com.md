---
type: "query"
date: "2026-10-06T21:30:43.607155+00:00"
question: "Fix wildcard whitelist rules such as *.spotify.com disappearing after Go server restart and audit similar blocklist persistence"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Engine", "Config", "DB", ".AddToWhitelist()", ".RemoveFromWhitelist()", ".AddToCustomBlocklist()", ".RemoveFromCustomBlocklist()", ".GetWhitelist()"]
---

# Q: Fix wildcard whitelist rules such as *.spotify.com disappearing after Go server restart and audit similar blocklist persistence

## Answer

Expanded query to whitelist, blocklist, wildcard, persist, config, database, save, reload, rule, engine. Found AddToWhitelist wrote SQLite but Engine.New loaded only config, while custom blocklist mutations were memory-only. Fixed startup to merge config plus persisted whitelist, added a dedicated persistent custom_blocklist table, made mutations persist before changing memory, surfaced database errors through APIs, and added database-reopen regression tests.

## Outcome

- Signal: useful

## Source Nodes

- Engine
- Config
- DB
- .AddToWhitelist()
- .RemoveFromWhitelist()
- .AddToCustomBlocklist()
- .RemoveFromCustomBlocklist()
- .GetWhitelist()