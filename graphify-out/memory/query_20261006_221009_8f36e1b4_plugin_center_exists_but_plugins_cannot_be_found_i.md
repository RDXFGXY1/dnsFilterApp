---
type: "query"
date: "2026-10-06T22:10:09.898960+00:00"
question: "Plugin Center exists but Plugins cannot be found in the dashboard navigation"
contributor: "graphify"
outcome: "useful"
source_nodes: ["dashboard.js"]
---

# Q: Plugin Center exists but Plugins cannot be found in the dashboard navigation

## Answer

Expanded query to navigation, sidebar, dashboard, plugin, system, render, group, view. Root cause: dashboard-navigation.js replaces the sidebar and only retains views listed in its group definitions; plugins was absent. Added plugins to the System group, bumped the navigation asset version to avoid stale browser cache, added a regression test, restarted the dev server, and verified the served script contains plugins.

## Outcome

- Signal: useful

## Source Nodes

- dashboard.js