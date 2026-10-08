---
type: "query"
date: "2026-10-05T20:26:19.069278+00:00"
question: "got a problem i want jist to do a goal and focuse so i got err"
contributor: "graphify"
outcome: "corrected"
correction: "Do not require a task selection; allow goal-only focus sessions and use task_id 0."
source_nodes: ["Goal", "Project", "Task", "Goal Project and Task Tracking"]
---

# Q: got a problem i want jist to do a goal and focuse so i got err

## Answer

Expanded via graph vocabulary: focus, goal, project, task, session, start. Corrected the Focus Engine client validation so a focus session can start from a goal alone, an optional project, or a specific task/sub-task. Goal-only sessions use task_id 0 and a descriptive Goal title, which the existing API and discipline manager already support.

## Outcome

- Signal: corrected
- Correction: Do not require a task selection; allow goal-only focus sessions and use task_id 0.

## Source Nodes

- Goal
- Project
- Task
- Goal Project and Task Tracking