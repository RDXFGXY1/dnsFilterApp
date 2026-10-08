---
type: "query"
date: "2026-10-08T00:37:48.210731+00:00"
question: "Implement Arch Linux App Network Control before real-machine testing"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Manager", "Filtering Controls", "netdns_linux.go"]
---

# Q: Implement Arch Linux App Network Control before real-machine testing

## Answer

Expanded from the prior graph vocabulary [linux, network, controls, platform, manager, block]. Added a Linux appnetwork platform using cgroup v2 for exact executable process grouping and a private nftables table for per-application outbound blocking. The manager restores stored rules, monitors newly launched matching processes, records original cgroups for safe Allow, exposes the runtime platform to the dashboard, provides reset-app-network for uninstall/recovery, and includes an Arch real-machine test checklist. Windows behavior remains behind its existing platform implementation.

## Outcome

- Signal: useful

## Source Nodes

- Manager
- Filtering Controls
- netdns_linux.go