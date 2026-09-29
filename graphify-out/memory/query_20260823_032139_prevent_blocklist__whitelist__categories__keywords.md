---
type: "query"
date: "2026-08-23T03:21:39.610117+00:00"
question: "Prevent Blocklist, Whitelist, Categories, Keywords and Apps lists from making the workspace too long"
contributor: "graphify"
outcome: "useful"
source_nodes: ["KeywordList", "WhitelistItem", "BlocklistsConfig", "CategoryManager", "dashboard.js"]
---

# Q: Prevent Blocklist, Whitelist, Categories, Keywords and Apps lists from making the workspace too long

## Answer

Added bounded 360px scrollable policy-list viewports, live visible/total item counts, compact density support, sticky search controls, and an explicit Expand list / Limit height toggle. Applied consistently to applications, categories, keyword groups, custom blocklist, and whitelist while keeping all workspace sections visible.

## Outcome

- Signal: useful

## Source Nodes

- KeywordList
- WhitelistItem
- BlocklistsConfig
- CategoryManager
- dashboard.js