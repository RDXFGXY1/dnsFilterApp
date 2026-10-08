---
type: "query"
date: "2026-10-06T21:18:16.005793+00:00"
question: "Create a README for Windows, Linux and Arch, plus an Arch desktop setup that enables everything with systemctl and starts automatically"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Cross-platform Runtime Guide", "netdns_linux.go", "setup.sh", "install.sh script", "System DNS Setup"]
---

# Q: Create a README for Windows, Linux and Arch, plus an Arch desktop setup that enables everything with systemctl and starts automatically

## Answer

Expanded from graph vocabulary: [linux, windows, service, desktop, setup, dns, system]. Added a complete Windows, Ubuntu/Debian, and Arch Linux installation/user guide; dedicated Arch build, install, and uninstall scripts; a systemd service enabled with enable --now; a Linux desktop menu entry; automatic package installation for .NET 8 and Avalonia native libraries; preserved configuration and data during upgrades/uninstall; and DNS setup/recovery documentation. Hardened Linux DNS routing to use active systemd-resolved only, otherwise NetworkManager, otherwise a backed-up resolv.conf. Built and inspected the final dns-filter-arch-x64.tar.gz archive successfully.

## Outcome

- Signal: useful

## Source Nodes

- Cross-platform Runtime Guide
- netdns_linux.go
- setup.sh
- install.sh script
- System DNS Setup