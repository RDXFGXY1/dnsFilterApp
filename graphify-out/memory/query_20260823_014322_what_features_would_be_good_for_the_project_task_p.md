---
type: "query"
date: "2026-08-23T01:43:22.955622+00:00"
question: "What features would be good for the project task planning workspace before implementing the big update?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Discipline Workspace", "Task", "Project", "loadWsTasks()", "taskRowHTML()", "addSubtask()", "loadCalendar()", "loadGoals()", "loadProjectActivity()"]
---

# Q: What features would be good for the project task planning workspace before implementing the big update?

## Answer

Expanded via graph vocabulary: [project, workspace, task, subtask, priority, status, labels, size, board, calendar, goals, analytics]. The existing workspace already supports task hierarchy, status, priority, labels, size, board/list/table modes, calendar integration, goals, and analytics. The recommended next design is progressive disclosure: a project command header, planning toolbar, backlog and milestones, task detail drawer, start/due dates, estimates and time tracking, dependencies and blockers, recurring tasks, saved filters, bulk editing, templates, timeline/calendar/workload views, risk and health summaries, and activity history. Advanced fields should be hidden in a task drawer rather than making every task row dense.

## Outcome

- Signal: useful

## Source Nodes

- Discipline Workspace
- Task
- Project
- loadWsTasks()
- taskRowHTML()
- addSubtask()
- loadCalendar()
- loadGoals()
- loadProjectActivity()