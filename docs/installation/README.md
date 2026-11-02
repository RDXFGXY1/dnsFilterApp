# Installation

This directory contains the official installation and recovery instructions for
the operating systems supported by DNS Filter.

The published Windows package is `v0.1.0`. The post-release features in the
[v0.2.0 draft](../releases/v0.2.0-draft.md) are source changes until a new
package passes the [release checklist](../RELEASE-CHECKLIST.md).

## Supported platforms

| Platform | Package | Current status |
|---|---|---|
| Windows 11 x64 | Portable ZIP and administrator installer | Verified |
| Arch and Arch-based Linux x64 | `.tar.gz`, systemd service, Avalonia desktop | DNS, dashboard, desktop, and boot startup verified on a real machine |
| Arch and Arch-based Linux ARM64 | `.tar.gz` source-build target | Build supported; real-device validation pending |

The Arch backend exposes the same dashboard and policy features as Windows.
Connection Freeze and App Network Control have native Linux implementations, but
their latest nftables/cgroup behavior remains in real-machine validation and must
not yet be treated as production-verified.

## Choose your platform

- [Windows installation](windows/README.md)
- [Arch Linux installation](arch/README.md)
- [Arch real-machine test checklist](arch/TEST-CHECKLIST.md)

## Shared architecture

Both packages contain two local components:

1. A privileged Go service provides DNS on port `53`, the dashboard/API on
   `127.0.0.1:8080`, filtering, storage, and operating-system controls.
2. An Avalonia desktop client connects to that service. Closing the desktop
   window does not stop DNS protection.

There are no default login credentials. First-run setup creates an administrator
account and a separate Secure Vault. Save both one-time recovery credentials
offline.

## Unsupported systems

Ubuntu, Debian, Fedora, macOS, Android, and iOS are not official installation
targets in this development phase. Do not describe a successful source compile
on one of those systems as an officially supported package.
