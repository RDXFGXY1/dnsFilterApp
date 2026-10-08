---
type: "query"
date: "2026-09-30T16:06:30.949766+00:00"
question: "Portable DNS Filter test commands failed"
contributor: "graphify"
outcome: "useful"
source_nodes: ["build-portable.ps1", "netdns_windows.go"]
---

# Q: Portable DNS Filter test commands failed

## Answer

Expanded through graph vocabulary: build, portable, start, dns, windows. The original Start batch was synchronous, so answering Yes to Terminate batch job stopped the server. The Enable batch then continued after setup-dns failed, disabled IPv6, and falsely printed Protection ENABLED. Fixed the portable Start launcher to spawn the server in the background and wait for matching-version health, fixed Enable to abort before changing adapters when DNS is not ready, and improved Disable to restore IPv6 while reporting IPv4 reset failure honestly. Rebuilt and audited the portable ZIP; host DNS resolution is working again.

## Outcome

- Signal: useful

## Source Nodes

- build-portable.ps1
- netdns_windows.go