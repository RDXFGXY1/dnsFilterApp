# Development Mode

Development mode runs DNS Filter directly from the repository. It does not create a portable ZIP and does not install or copy a release into Program Files.

## Start or rebuild and restart

Double-click `dev-server.bat`, or run this from PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1 restart
```

Administrator permission is required because the DNS server binds port 53. The command builds `build\dev\dns-filter-dev.exe`, stops only DNS Filter-owned processes holding the application ports, and starts the development executable with the repository as its working directory.

## Which changes require a restart?

| Changed files | Required action |
| --- | --- |
| `web/static/css/*` or `web/static/js/*` | Refresh the browser. Use `Ctrl+F5` if cached. |
| `web/templates/*` | Restart the development server because Go parses templates at startup. |
| Go files under `cmd/`, `internal/`, or `pkg/` | Rebuild and restart with the command above. |
| Configuration under `configs/` | Restart unless the specific setting has a live API control. |

## Other commands

```powershell
# Restart an already-built executable without rebuilding Go:
.\scripts\dev-server.ps1 restart -NoBuild

# Stop development mode:
.\scripts\dev-server.ps1 stop

# Show status and health:
.\scripts\dev-server.ps1 status
```

## Data and logs

Development mode keeps using the repository's existing `configs\` and `data\` directories. This preserves the local account, Vault, database, journal, focus sessions, and logs.

Runner output is written to:

- `data\logs\dev-server.stdout.log`
- `data\logs\dev-server.stderr.log`

Do not run the portable/installed server and the development server simultaneously; they require the same DNS and dashboard ports.
