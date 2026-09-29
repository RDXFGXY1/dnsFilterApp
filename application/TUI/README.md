# DNS Filter Console

A single-executable Windows control center for DNS Filter. The console owns the DNS engine, localhost dashboard, Windows DNS routing, startup registration, and system-tray lifecycle while presenting trustworthy protection analytics.

Detailed filtering rules remain in the hosted web dashboard at `http://127.0.0.1:8080`.

## Application controls

- First-run guided setup and DNS activation.
- Start or stop the complete DNS Filter service.
- Pause or resume filtering while keeping the dashboard available.
- Route Windows DNS through the local filter or restore normal DHCP DNS.
- Enable elevated launch at Windows login.
- Start minimized and close the console into the system tray.
- Tray controls for console, dashboard, service, filtering, DNS recovery, and safe exit.
- Keyboard and mouse navigation.

## Views

- **Overview** — protection score, 24-hour metrics, activity sparkline, health checks, and latest events.
- **Traffic** — automatically refreshed blocked DNS requests with client and timestamp details.
- **Threats** — ranked blocked domains with proportional activity bars.
- **Policy** — enabled category coverage and trusted whitelist inventory.
- **System** — filtering state, service ports, upstream DNS, cache, routing interface, and health checks.

## Controls

| Key | Action |
|---|---|
| `1`–`5` | Open a monitoring view |
| `Tab`, `←`, `→` | Move between views |
| `↑`, `↓`, `J`, `K` | Navigate rows |
| `R` | Refresh analytics immediately |
| `Space` | Pause or resume automatic refresh |
| `O` | Open the localhost dashboard to make changes |
| `F1`, `?` | Open help |
| `S` | Open server and application settings |
| `Q` | Hide in the tray when Close to tray is enabled |
| `Ctrl+C` | Stop, restore DNS, and exit |

## Build and run

```powershell
cd application\TUI
go build -o dns-filter-console.exe .
.\dns-filter-console.exe
```

The default server is `http://127.0.0.1:8080`. A different local URL can be passed as the first argument.

The console uses `DNS_FILTER_USERNAME` and `DNS_FILTER_PASSWORD` when set. Otherwise it uses the application's development defaults.

## Architecture

```text
application/TUI/
├── main.go       program entrypoint
├── model.go      application state and initialization
├── update.go     keyboard and refresh state machine
├── service.go    in-process DNS and dashboard lifecycle
├── settings.go   persistent setup and startup preferences
├── controls.go   service actions and settings behavior
├── tray.go       Windows tray menu and lifecycle
├── window_*.go   console hide/show integration
├── tray_icon.go  generated embedded application icon
├── api.go        authenticated analytics client
├── view.go       complete responsive interface
├── theme.go      colors and visual styles
├── types.go      API and view data structures
├── messages.go   Bubble Tea messages
└── utils.go      formatting and chart helpers
```

## Configuration boundary

The console controls application lifecycle, filtering state, Windows startup, and DNS recovery. Domain rules, categories, blocklists, AI inspection, and other detailed policies remain in the hosted dashboard. Press `O` to open it.
