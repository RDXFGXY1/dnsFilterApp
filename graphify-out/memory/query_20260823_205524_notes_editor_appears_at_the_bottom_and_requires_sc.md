---
type: "query"
date: "2026-08-23T20:55:24.087182+00:00"
question: "Notes editor appears at the bottom and requires scrolling after pressing New"
contributor: "graphify"
outcome: "corrected"
correction: "Open the editor in place on desktop and as a responsive overlay on smaller screens instead of leaving it below the notebook list."
source_nodes: ["DNS Filter Management Dashboard", "dashboard.js"]
---

# Q: Notes editor appears at the bottom and requires scrolling after pressing New

## Answer

The responsive notes grid stacked the editor below the notebook. Fixed dashboard-organizer.js, dashboard-organizer.css, and dashboard.html so New immediately opens and focuses the title, desktop editor stays visible, and screens under 950px use a closeable full-viewport editor overlay with scroll locked behind it.

## Outcome

- Signal: corrected
- Correction: Open the editor in place on desktop and as a responsive overlay on smaller screens instead of leaving it below the notebook list.

## Source Nodes

- DNS Filter Management Dashboard
- dashboard.js