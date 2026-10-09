---
type: "query"
date: "2026-10-09T21:05:29.515660+00:00"
question: "Why does Connection Freeze fail on Arch Linux, and how do the API, Linux nftables backend, systemd service, and UI connect?"
contributor: "graphify"
outcome: "dead_end"
source_nodes: ["Server", "api"]
---

# Q: Why does Connection Freeze fail on Arch Linux, and how do the API, Linux nftables backend, systemd service, and UI connect?

## Answer

Expanded from original query via graph vocab: [linux, network, platform, table, output, system, service, api]. The existing graph was stale and did not contain the current Connection Freeze Linux backend, so direct source inspection was required. The backend installs a private inet dnsfilter_freeze output chain, the API calls Manager.Start, systemd runs the service as root and cleans the table on stop, and the dashboard calls the connection-freeze API.

## Outcome

- Signal: dead_end

## Source Nodes

- Server
- api