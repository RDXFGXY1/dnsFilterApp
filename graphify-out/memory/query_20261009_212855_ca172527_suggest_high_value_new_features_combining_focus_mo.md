---
type: "query"
date: "2026-10-09T21:28:55.597410+00:00"
question: "Suggest high-value new features combining focus mode, Brave tab locking, and allowing network only for selected applications"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Engine", ".StartFocus()", ".SetAppBlocked()", ".NoteActivity()"]
---

# Q: Suggest high-value new features combining focus mode, Brave tab locking, and allowing network only for selected applications

## Answer

Expanded from original query via graph vocab: [focus, app, network, process, profile, block, domain, activity, schedule, task]. The existing Engine already links focus state, app-domain policies, blocked-domain decisions, and activity tracking. The best next architecture is a Focus Capsule: operating-system app allowlisting plus a Brave extension that controls tabs by browser tab ID and URL rules, not renderer PID, because Chromium networking is shared. Add safe override, crash recovery, essential-service exemptions, session contracts, distraction-free page transformations, temporary access requests, network/time budgets, session replay, and reusable signed workflow profiles.

## Outcome

- Signal: useful

## Source Nodes

- Engine
- .StartFocus()
- .SetAppBlocked()
- .NoteActivity()