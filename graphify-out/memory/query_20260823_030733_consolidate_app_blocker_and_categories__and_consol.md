---
type: "query"
date: "2026-08-23T03:07:33.423285+00:00"
question: "Consolidate App Blocker and Categories, and consolidate Blocklist, Whitelist, and AI Blocker into organized tabs"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Filtering Controls", "dashboard.js", "AIBlocker", "CategoryManager", "KeywordManager", "WhitelistItem"]
---

# Q: Consolidate App Blocker and Categories, and consolidate Blocklist, Whitelist, and AI Blocker into organized tabs

## Answer

Expanded from original query via graph vocab: dashboard, switch, view, filtering, apps, categories, category, keyword, blocklist, whitelist, blocker, filter. The dashboard filtering UI was consolidated into Protection (Applications, Categories, Keywords) and Rules & Intelligence (Blocklist, Whitelist, AI decisions). Existing loaders and API controls were preserved, legacy deep links route into the correct workspace pane, workspace metrics and persistent sub-tab selection were added.

## Outcome

- Signal: useful

## Source Nodes

- Filtering Controls
- dashboard.js
- AIBlocker
- CategoryManager
- KeywordManager
- WhitelistItem