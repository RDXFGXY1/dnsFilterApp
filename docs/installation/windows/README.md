# Install DNS Filter on Windows

## Status

Windows 11 x64 is the verified Windows target. Installation and system controls
require Administrator permission.

## Requirements for building from source

- Go matching the version declared in `go.mod`.
- .NET 8 SDK.
- Windows PowerShell.
- Administrator access for installation, port 53, DNS routing, firewall, and
  other system controls.

## Build the portable release

From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-portable.ps1
```

Outputs:

- `dist\dns-filter-portable\`
- `dist\dns-filter-portable-<version>.zip`

The release builder compiles the Go service, publishes the Windows Avalonia
client, copies clean configuration and web assets, creates maintenance shortcuts,
and runs the release audit.

## Install or update

1. Extract the ZIP into a temporary directory.
2. Run **Install or Update DNS Filter.bat** as Administrator.
3. Open `%ProgramFiles%\DNS Filter\desktop\DNSFilter.exe`.
4. Complete first-run setup for a new installation.
5. Enable system DNS routing from the desktop client or dashboard.

An update preserves configuration, database, Vault, plugins, certificates,
recovery state, and logs. If the new service fails its health check, the updater
rolls back the replaced executable and web assets.

## Development mode

Use repository files directly without creating a portable ZIP:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

Restart the development server after Go changes. Dashboard HTML, CSS, and
JavaScript changes become visible after a browser refresh.

## Health and recovery

If DNS Filter cannot start or `dns-filter.exe` is unavailable, run
**EMERGENCY - Restore Normal DNS.bat** as Administrator from the extracted or
installed release folder. It stops DNS Filter temporarily and restores automatic
DNS without deleting configuration, Vault, logs, or other user data. Automatic
startup is paused; run **Install or Update DNS Filter.bat** again when you want
to re-enable protection.

Run commands from an Administrator PowerShell terminal:

```powershell
# API health
Invoke-RestMethod http://127.0.0.1:8080/healthz

# Route Windows DNS through DNS Filter
.\dns-filter.exe setup-dns

# Restore automatic system DNS
.\dns-filter.exe reset-dns

# Release application firewall, quarantine, and schedule rules
.\dns-filter.exe reset-app-network
```

## Uninstall

Run **Uninstall DNS Filter.bat** as Administrator. Normal uninstall restores
DNS Filter-owned system changes and preserves user data. Destructive data removal
requires the explicit confirmation requested by the uninstaller.
