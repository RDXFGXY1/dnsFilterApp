---
type: "query"
date: "2026-08-23T03:48:00.735638+00:00"
question: "Add dark mode, a Discipline weekly timetable, and tiny local AI site analysis"
contributor: "graphify"
outcome: "useful"
source_nodes: ["AIBlocker", "DomainFeatures", "loadCalendar()", "ContentInspector", "discipline.js"]
---

# Q: Add dark mode, a Discipline weekly timetable, and tiny local AI site analysis

## Answer

Implemented persistent Light/Dark/System themes across Dashboard and Discipline, a localStorage-backed seven-day timetable with colored recurring time blocks in Discipline Calendar, and an authenticated explainable domain-risk analyzer using the existing local Naive Bayes model. The analyzer exposes confidence, reason, feature signals, and rule-preparation actions while explicitly limiting this phase to DNS-visible domain signals; encrypted page metadata requires a later browser/proxy layer.

## Outcome

- Signal: useful

## Source Nodes

- AIBlocker
- DomainFeatures
- loadCalendar()
- ContentInspector
- discipline.js