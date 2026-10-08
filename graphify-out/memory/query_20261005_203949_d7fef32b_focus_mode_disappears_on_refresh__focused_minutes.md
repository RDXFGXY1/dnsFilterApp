---
type: "query"
date: "2026-10-05T20:39:49.298287+00:00"
question: "focus mode disappears on refresh, focused minutes stays zero, and hide setup while focusing"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Discipline Workspace", "api/discipline.go", "discipline/discipline.go", "Manager"]
---

# Q: focus mode disappears on refresh, focused minutes stays zero, and hide setup while focusing

## Answer

Expanded via graph vocabulary: focus, session, active, restore, timer, local, storage, activity, history. Added an authenticated active-session API backed by Manager.ActiveFocusSessionID, persisted countdown/running state in browser storage, restored live sessions after page refresh, updated today's focused counter every second, switched the running UI to timer plus Pause/Resume and Restart controls, and added a compact live activity summary. Server restarts now close stale active sessions as cancelled so history has no ghost sessions.

## Outcome

- Signal: useful

## Source Nodes

- Discipline Workspace
- api/discipline.go
- discipline/discipline.go
- Manager