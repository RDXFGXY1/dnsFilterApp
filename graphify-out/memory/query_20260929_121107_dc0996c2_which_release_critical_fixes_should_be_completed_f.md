---
type: "query"
date: "2026-09-29T12:11:07.149060+00:00"
question: "Which release-critical fixes should be completed first before the official release?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Unreleased Changes", "Release Changelog", "build-portable.ps1", "github.com/RDXFGXY1/dns-filter-app"]
---

# Q: Which release-critical fixes should be completed first before the official release?

## Answer

Release verification was the first critical batch: align CI with go.mod Go 1.25, test Go on Linux and Windows, build the .NET 8 WPF desktop, validate dashboard JavaScript, remove the invalid static Pages deployment, update authentication documentation and UNRELEASED notes, add a release checklist, and format legacy Go files so the new gate passes. Local Go tests, WPF Release build, JavaScript syntax, gofmt, and diff checks passed.

## Outcome

- Signal: useful

## Source Nodes

- Unreleased Changes
- Release Changelog
- build-portable.ps1
- github.com/RDXFGXY1/dns-filter-app