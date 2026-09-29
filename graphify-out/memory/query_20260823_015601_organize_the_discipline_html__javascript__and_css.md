---
type: "query"
date: "2026-08-23T01:56:01.635883+00:00"
question: "Organize the Discipline HTML, JavaScript, and CSS before building the new project-planning features one by one"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Discipline Workspace", "api()", "Project", "Task", "Server", "loadWsTasks()", "taskRowHTML()"]
---

# Q: Organize the Discipline HTML, JavaScript, and CSS before building the new project-planning features one by one

## Answer

Expanded via graph vocabulary: [discipline, template, server, api, workspace, task, project, html]. Split the Discipline monolith into six Gin template partials plus a 38-line shell, eight ordered classic JavaScript feature modules, and a separate Discipline stylesheet while preserving shared global behavior. Verified every Discipline view and project mode. Then implemented feature 1: persistent Project Command Header with objective, lifecycle status, start and target dates, goal context, progress, open-work count, calculated health, database migration, and PATCH project API.

## Outcome

- Signal: useful

## Source Nodes

- Discipline Workspace
- api()
- Project
- Task
- Server
- loadWsTasks()
- taskRowHTML()