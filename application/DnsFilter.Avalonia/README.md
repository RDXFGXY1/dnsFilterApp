# DNS Filter cross-platform desktop

This is the shared Avalonia desktop client for Windows and Linux. It connects only
to the local Go service at `http://127.0.0.1:8080` and keeps the authenticated
session cookie inside the desktop process.

## Current shared features

- native dark interface with no browser redirect;
- local login and session handling;
- live blocked-query activity and top-domain analytics;
- filtering enable/disable, blocklist update, and cache clear;
- system DNS setup and reset through the Go service;
- protection-health checks;
- capability-aware hard-security and Connection Freeze screens.

Windows exposes the existing Firewall/WFP controls. Linux displays those controls
as unavailable until their nftables and process-telemetry backends are implemented;
the application never pretends that a protection is active when it is unsupported.

## Development

```text
dotnet run --project application/DnsFilter.Avalonia/DnsFilter.Avalonia.csproj
```

## Publish

```text
dotnet publish application/DnsFilter.Avalonia/DnsFilter.Avalonia.csproj -c Release -r win-x64 --self-contained false
dotnet publish application/DnsFilter.Avalonia/DnsFilter.Avalonia.csproj -c Release -r linux-x64 --self-contained false
```

The machine needs the .NET 8 runtime for framework-dependent packages.
