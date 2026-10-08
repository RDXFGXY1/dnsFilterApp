---
type: "query"
date: "2026-10-08T00:24:24.137081+00:00"
question: "Implement the next Arch Linux native network-control phase, starting with Connection Freeze and then App Network Control"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Manager", "netdns_linux.go", "Filtering Controls"]
---

# Q: Implement the next Arch Linux native network-control phase, starting with Connection Freeze and then App Network Control

## Answer

Expanded from original query via graph vocab: [linux, network, controls, platform, manager, block]. The graph showed platform-specific backends behind the existing managers, so the Linux implementation was added behind connectionfreeze's backend interface without changing the API or UI. It uses a private nftables table, preserves established flows and essential DNS/DHCP/ICMP traffic, records aggregate dropped-packet events, and removes only its own table on stop.

## Outcome

- Signal: useful

## Source Nodes

- Manager
- netdns_linux.go
- Filtering Controls