---
type: "query"
date: "2026-10-05T20:28:39.854726+00:00"
question: "f3lt 5ayar diyal local activy log fen n9r nchouf logs?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Discipline Workspace", "discipline/discipline.go", "Manager"]
---

# Q: f3lt 5ayar diyal local activy log fen n9r nchouf logs?

## Answer

Expanded via graph vocabulary: focus, activity, log, session, history, details, discipline. Focus activity is recorded locally in data/dns-filter.db (discipline_focus_sessions and discipline_focus_activity). The backend exposes /api/discipline/focus-sessions and /api/discipline/focus-sessions/:id, but the current Discipline UI has no detailed session-history panel; the Journal only receives a completion summary.

## Outcome

- Signal: useful

## Source Nodes

- Discipline Workspace
- discipline/discipline.go
- Manager