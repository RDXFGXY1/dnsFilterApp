---
type: "query"
date: "2026-08-23T21:01:44.806267+00:00"
question: "Add project YAML import and export for AI-generated plans"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Discipline Workspace", "Project", "discipline.js"]
---

# Q: Add project YAML import and export for AI-generated plans

## Answer

Implemented full project YAML transfer using project_transfer.go, project-transfer.js, discipline-projects.tmpl, and discipline.css. The version-1 schema preserves project metadata, milestones, cycles, tasks, subtasks, checklists, backlog/archive state, priorities, estimates, links, and key-based dependencies. Added strict preview validation, upload/paste, AI template/prompt, add/replace-matching/skip modes, export-all and per-project export, safe transactions, and parser tests.

## Outcome

- Signal: useful

## Source Nodes

- Discipline Workspace
- Project
- discipline.js