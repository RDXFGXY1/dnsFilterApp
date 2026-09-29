# Windows Desktop Application

## Purpose

The C# WPF application is a native Windows control center connected to the same Go service. It is not a second filtering engine and does not duplicate policy logic.

## Authentication

The password is sent only to the local Go service. The client stores its session cookie in its private HTTP client and does not save the password.

## Native overview

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

Shows high-risk unseen destinations and can create Windows balloon notifications. Clicking a notification brings the native application to the relevant section rather than opening a web browser.

## Download Guardian

Shows download analytics, live files, signatures, hashes, risk explanations, quarantine, and restore inside the desktop window.

## Dark-only interface

Uses a pure-black application background, dark Windows title bar, dark tables, high-contrast text, status colors, and responsive scrolling.

## Local-service startup

When the service is unavailable, the desktop app searches expected nearby build locations and attempts to start it hidden. Administrator privileges are required for several DNS and firewall capabilities.

## Packaging

The self-contained Windows x64 build packages the .NET runtime and can run without a separate .NET installation. The Go service and its web/config/data directories remain required.

