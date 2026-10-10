---
type: "query"
date: "2026-10-09T21:39:29.374448+00:00"
question: "Design a Focus Capsule that allows selected applications, permits YouTube videos from the selected channel, shows a custom block page for distractions, and includes AI Session Coach and Focus Recovery"
contributor: "graphify"
outcome: "useful"
source_nodes: [".persistBlockedApps()", ".SetAppBlocked()", ".StartFocus()", "BlockPageServer", "Goal Project and Task Tracking"]
---

# Q: Design a Focus Capsule that allows selected applications, permits YouTube videos from the selected channel, shows a custom block page for distractions, and includes AI Session Coach and Focus Recovery

## Answer

Expanded from original query via graph vocab: [focus, app, apps, network, process, profile, block, blockpage, domain, activity, task, goal]. The existing Engine already connects persisted app controls, StartFocus/FocusStatus, domain policy, BlockPageServer, and Discipline goal/task tracking. Implement the new capability as a Focus Capsule coordinating those existing primitives with a Brave extension. The extension should identify YouTube channel IDs from the initial video and allow later watch pages only when their channel ID matches; it must not rely on browser process IDs. Add selectable block-page personalities, emergency/temporary access, crash recovery, and an AI Session Coach using locally collected session events.

## Outcome

- Signal: useful

## Source Nodes

- .persistBlockedApps()
- .SetAppBlocked()
- .StartFocus()
- BlockPageServer
- Goal Project and Task Tracking