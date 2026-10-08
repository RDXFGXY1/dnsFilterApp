# Cross-Platform Desktop Application

## Purpose

The C# Avalonia application is a dark desktop control center for Windows and Arch Linux. It connects to the same local Go service, so it is not a second filtering engine and does not duplicate policy logic.

## Authentication

The password is sent only to the local Go service. The client stores its session cookie in its private HTTP client and does not save the password.

## Desktop overview

Shows filtering state, system DNS, 24-hour blocks, rule coverage, protection health, timeline, top domains, and recent blocked activity.

## Direct controls

- Enable or disable filtering.
- Configure or restore system DNS.
- Clear cache.
- Update blocklists.
- Start or stop Connection Freeze.
- Enable or disable DNS Bypass Kill Switch.

## Native investigation

The domain inspector displays decision, winning layer, reason, history, recommendation, and quick block/allow actions.

## Hard Security status

Displays DNS bypass enforcement and the number of quarantined applications.

## New Connections

Shows high-risk unseen destinations inside the desktop application. Supported desktop notifications bring the application to the relevant section rather than opening the web dashboard.

## Download Guardian

Shows download analytics, live files, signatures, hashes, risk explanations, quarantine, and restore inside the desktop window.

## Dark-only interface

Uses a pure-black application background, dark window chrome where supported, dark tables, high-contrast text, status colors, and responsive scrolling.

## Local-service startup

When the service is unavailable, the desktop app can connect to or start the local service from supported installation layouts. Administrator/root privileges are required for system DNS and native network controls.

## Packaging

Windows and Arch packages include the desktop application beside the Go service and its web/config/data layout. Packaging may be self-contained for a target architecture, while the Go service remains the source of truth for filtering and policy.

The older WPF client remains historical Windows-specific code; Avalonia is the maintained cross-platform desktop direction.

