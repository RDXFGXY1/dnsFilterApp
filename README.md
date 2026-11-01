# DNS Filter

DNS Filter is a local-first DNS protection, network-control, analytics, and
productivity application. A privileged Go service provides DNS filtering and a
local API; the same service hosts the web dashboard, while an Avalonia desktop
client provides a native interface on Windows and Linux.

**Release status:** v0.1.0 is the current version in source. The work since the
public baseline is summarized in the [v0.2.0 draft](docs/releases/v0.2.0-draft.md).
That draft describes the working tree; it is not a published package.
See [the post-v0.1.0 changelog](docs/releases/UNRELEASED.md) for the itemized
additions, changes, fixes, and release limitations.

## Supported systems

| System | Status | Package and service |
|---|---|---|
| Windows 11 x64 | Validated development platform | Portable package, Windows background startup, Avalonia desktop |
| Arch Linux / Arch-based x64 | DNS, dashboard, desktop, and reboot startup verified; advanced network-control validation in progress | Arch archive, `systemd`, Avalonia desktop, `nftables` and cgroup v2 |
| Arch Linux / Arch-based ARM64 | Build target available; real-machine validation pending | Arch ARM64 archive and `systemd` |
| Other Linux distributions and macOS | Not officially supported in this phase | Core code may compile, but installation and native security controls are not supported |

Start with the [installation overview](docs/installation/README.md), then choose
the dedicated [Windows](docs/installation/windows/README.md) or
[Arch Linux](docs/installation/arch/README.md) guide.

## Main features

- DNS filtering with categories, keywords, direct block rules, wildcard
  exceptions, schedules, and encrypted upstream DNS.
- Live activity, analytics, decision explanations, connection intelligence, and
  locally stored logs.
- Security Center, Secure Vault, encrypted backups, recovery credentials,
  emergency lockdown, signed security packs, Download Guardian, and Hard System.
- Connection Freeze on Windows and Arch Linux.
- Per-application network control and quarantine through Windows Firewall or
  Linux cgroup v2 plus `nftables`.
- Plugin Center with API-only plugins, managed local runtimes, native controls,
  sandboxed local web views, and the TinyPrinter integration.
- Discipline workspace with goals, projects, nested tasks, focus sessions,
  opt-in activity logging, Daily Journal, habits, reminders, analytics, and the
  Intelligence Inbox.
- Focus Capsules that combine selected goals or tasks with allowed applications,
  websites, and YouTube channels, a local block page, crash recovery, and an
  optional AI Session Coach review.
- Dark Avalonia desktop application and a full local web dashboard.
- Installable `.adnf` Side Apps with isolated HTML workspaces; Focus OS is the
  example package. See the [Side Apps guide](docs/SIDE-APPS.md).
- Windows startup before sign-in, a tray icon, and deadline notifications. See
  [startup](docs/WINDOWS-STARTUP.md) and [notifications](docs/WINDOWS-NOTIFICATIONS.md).
- Dashboard controls for local domain names and limited LAN access, a saved
  200-entry Live view, DNS health timeline, and Network Test history.
- Vault recovery through a configured SMTP mailbox or USB recovery material.

## Architecture

1. The Go service listens on DNS port `53`, hosts the local dashboard/API on
   `127.0.0.1:8080`, evaluates policy, and stores local state in SQLite.
2. The Avalonia desktop application connects to that local API. Closing the
   desktop window does not stop DNS protection.
3. Windows native controls use Windows Firewall/WFP. Arch native controls use
   private `nftables` tables and cgroup v2.

The application is local-first. Some optional features, such as configured AI
providers or third-party plugins, can contact services chosen by the user.

## Windows quick start

Requirements for building: Go, .NET 8 SDK, and Windows PowerShell.

```powershell
git clone https://github.com/RDXFGXY1/dnsFilterApp.git
cd dnsFilterApp
powershell -ExecutionPolicy Bypass -File .\scripts\build-portable.ps1
```

The build creates `dist\dns-filter-portable` and a versioned ZIP. Open the
portable folder and run **Install or Update DNS Filter.bat** as Administrator.
Then open `desktop\DNSFilter.exe` or visit `http://127.0.0.1:8080`.

For repository development without creating a portable ZIP:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

## Arch Linux quick start

Requirements for building: Go, .NET 8 SDK, `bash`, and standard archive tools.

```bash
git clone https://github.com/RDXFGXY1/dnsFilterApp.git
cd dnsFilterApp
chmod +x scripts/build-arch.sh
./scripts/build-arch.sh x64
cd dist
tar -xzf dns-filter-arch-x64.tar.gz
cd dns-filter-linux-x64
sudo ./install-arch.sh
```

The installer verifies Arch, installs runtime dependencies, installs the service
under `/opt/dns-filter`, enables `dns-filter.service`, waits for the health check,
and then routes system DNS through the local filter.

Run diagnostics after installation:

```bash
sudo /opt/dns-filter/arch-diagnostics.sh
```

Open **DNS Filter** from the application menu, run
`/opt/dns-filter/desktop/DNSFilter`, or visit `http://127.0.0.1:8080`.

The first real-machine validation sequence is documented in
[docs/installation/arch/TEST-CHECKLIST.md](docs/installation/arch/TEST-CHECKLIST.md).

## First-run setup

There is no default username or password. On first launch, the local setup page
requires you to:

1. Create an administrator username and strong password.
2. Create or reuse that password for the Secure Vault.
3. Choose the Vault auto-lock time.
4. Save the separate one-time account recovery key and Vault recovery code.

The setup route is loopback-only and closes after setup succeeds. Store both
recovery credentials offline; they protect different layers and cannot be shown
again.

## Everyday operation

- Dashboard: `http://127.0.0.1:8080`
- Local health: `http://127.0.0.1:8080/healthz`
- Enable system DNS routing: `dns-filter setup-dns`
- Restore automatic DNS: `dns-filter reset-dns`
- Release all per-app network rules: `dns-filter reset-app-network`

On Windows, run privileged maintenance commands from an Administrator PowerShell.
On Arch, use `sudo /opt/dns-filter/dns-filter <command>`.

## Updating and removing

Windows:

- Run **Install or Update DNS Filter.bat** from the new package.
- Run **Uninstall DNS Filter.bat** to remove it.
- Updates and normal uninstall preserve configuration and local data by default.

Arch:

```bash
sudo ./install-arch.sh          # install or update
sudo ./uninstall-arch.sh        # uninstall and preserve local data
sudo ./uninstall-arch.sh --purge
```

Arch updates create timestamped backups in `/var/lib/dns-filter-backups`.
A normal uninstall preserves state in `/var/lib/dns-filter-preserved`; `--purge`
removes preserved data and backups.

## Development and verification

```bash
go test ./...
go run ./cmd/server -dev
```

Platform packages:

```powershell
# Windows
powershell -ExecutionPolicy Bypass -File .\scripts\build-portable.ps1
```

```bash
# Arch x64 or ARM64
./scripts/build-arch.sh x64
./scripts/build-arch.sh arm64
```

Do not commit or publish active `configs/` or `data/` directories. They can
contain password hashes, Vault state, recovery data, plugin configuration, and
local browsing/activity history.

## Documentation

- [Installation overview](docs/installation/README.md)
- [Windows installation](docs/installation/windows/README.md)
- [Arch Linux installation](docs/installation/arch/README.md)
- [Documentation index](docs/README.md)
- [Arch test checklist](docs/installation/arch/TEST-CHECKLIST.md)
- [Development mode](docs/DEVELOPMENT-MODE.md)
- [Browser extension setup](extensions/hard-system/README.md)
- [Plugin SDK](docs/PLUGIN-SDK.md)
- [Feature documentation](docs/features/README.md)
- [Releases and changelog](docs/releases/README.md)
- [v0.2.0 draft and comparison with the public baseline](docs/releases/v0.2.0-draft.md)
- [Unreleased changes](docs/releases/UNRELEASED.md)
- [v0.1.0 first public release notes](docs/releases/v0.1.0.md)

## Important limitations

- DNS Filter improves policy control and visibility but does not replace an
  antivirus product, operating-system updates, a firewall strategy, or a VPN.
- DNS filtering sees domain lookups, not the full contents of encrypted traffic.
- Browser Secure DNS can bypass system DNS unless it is disabled or configured to
  use the local filter.
- Arch App Network Control is new and must be validated on real hardware. Test it
  with a non-critical application before relying on it.

## License
DNSFilter is free and open-source software licensed under the **GNU General Public License v3.0 (GPL-3.0-only)**.
You are free to use, study, modify, and redistribute the software under the terms of the GPLv3.
See the [LICENSE](LICENSE) file for details.
The DNSFilter name and official branding are subject to a separate [Trademark Policy](TRADEMARKS.md).
