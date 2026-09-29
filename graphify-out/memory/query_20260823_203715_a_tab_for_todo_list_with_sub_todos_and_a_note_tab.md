---
type: "query"
date: "2026-08-23T20:37:15.966847+00:00"
question: "a tab for todo list with sub-todos and a note tab"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", ".setupRoutes()", "New()", "dashboard.js", "addSubtask()"]
---

# Q: a tab for todo list with sub-todos and a note tab

## Answer

Added persistent dashboard organizer APIs through Server route registration and SQLite tables, plus dashboard Todo and Notes views. Todo supports nested sub-todos, filters, search, progress, editing, cascading completion and recursive deletion. Notes supports creation, search, autosave, pinning, word counts and deletion.

## Outcome

- Signal: useful

## Source Nodes

- Server
- .setupRoutes()
- New()
- dashboard.js
- addSubtask()