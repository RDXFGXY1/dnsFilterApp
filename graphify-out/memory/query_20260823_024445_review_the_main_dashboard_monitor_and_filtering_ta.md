---
type: "query"
date: "2026-08-23T02:44:45.262927+00:00"
question: "Review the main dashboard Monitor and Filtering tabs and propose hidden ideas before editing"
contributor: "graphify"
outcome: "useful"
source_nodes: ["dashboard.js", "Stats", "AIBlocker", "FilteringConfig", "CategoryManager", "KeywordManager", "WhitelistItem"]
---

# Q: Review the main dashboard Monitor and Filtering tabs and propose hidden ideas before editing

## Answer

Expanded from graph vocabulary: dashboard, live, activity, focus, filtering, blocker, categories, keywords, blocklist, whitelist, requests, stats. Traversal shows dashboard.html/dashboard.js as the UI hub connected to stats, recent queries, category and keyword managers, filter engine, AI blocker, whitelist, focus, and unblock requests. Recommended reorganizing the dashboard into Observe, Analyze, Control, Exceptions, and Automate layers; adding investigation drawers and rule traces to Live/Activity; policy profiles, schedules, previews, conflicts and audit history to Filtering; and posture, incidents, anomaly, latency, client and trend intelligence to Monitor.

## Outcome

- Signal: useful

## Source Nodes

- dashboard.js
- Stats
- AIBlocker
- FilteringConfig
- CategoryManager
- KeywordManager
- WhitelistItem