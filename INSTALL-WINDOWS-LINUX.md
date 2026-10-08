# DNS Filter installation guide

This guide covers the two official targets for the current development phase:
Windows 11 x64 and Arch/Arch-based Linux. Arch packages compile successfully,
but native Linux security controls still require real-machine validation.

## How the application runs

DNS Filter has two local components:

1. A privileged Go service provides DNS on port `53`, the dashboard/API on
   `127.0.0.1:8080`, filtering, storage, and operating-system controls.
2. An Avalonia desktop client connects to the local service. Closing the client
   does not stop protection.

The first launch has no default credentials. Open the local dashboard, create an
administrator and Secure Vault, then save both one-time recovery credentials.

## Windows 11 x64

### Requirements for a source build

- Go matching the version declared in `go.mod`.
- .NET 8 SDK.
- Windows PowerShell.
- Administrator permission for installation, port 53, DNS routing, firewall, and
  other system controls.

### Build

From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build-portable.ps1
```

Outputs:

- `dist\dns-filter-portable\`
- `dist\dns-filter-portable-<version>.zip`

The release builder creates the Go service, publishes the Windows Avalonia client,
copies clean configuration and web assets, creates launch/maintenance shortcuts,
and runs the release audit.

### Install or update

1. Extract the ZIP into a temporary folder.
2. Right-click **Install or Update DNS Filter.bat** and run it as Administrator.
3. Open `%ProgramFiles%\DNS Filter\desktop\DNSFilter.exe`.
4. Complete first-run setup if this is a new installation.
5. Enable local DNS from the desktop client or dashboard.

The updater health-checks the new service and rolls the executable/web assets back
if startup fails. Configuration, database, Vault, certificates, plugins, logs, and
recovery state are preserved.

### Development mode

To use repository files directly without building a portable ZIP:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

Restart the development server after Go changes. Dashboard HTML/CSS/JavaScript
changes are read from the repository and become visible after a browser refresh.

### Windows maintenance

```powershell
# Health
Invoke-RestMethod http://127.0.0.1:8080/healthz

# Route Windows DNS through the filter
.\dns-filter.exe setup-dns

# Restore automatic DNS
.\dns-filter.exe reset-dns

# Release all application firewall/quarantine/schedule rules
.\dns-filter.exe reset-app-network
```

Run **Uninstall DNS Filter.bat** as Administrator to remove the installation.
Normal removal preserves user configuration and data unless destructive removal is
explicitly selected.

## Arch Linux and Arch-based distributions

The dedicated installer accepts Arch and distributions that declare Arch through
`ID` or `ID_LIKE`, use `pacman`, and run `systemd`.

### Requirements for a source build

- Go matching `go.mod`.
- .NET 8 SDK.
- `bash`, `tar`, and normal build utilities.

Runtime dependencies such as .NET 8, Avalonia native libraries, `curl`, and
`nftables` are installed by the Arch installer unless `--skip-deps` is used.

### Build

```bash
chmod +x scripts/build-linux.sh scripts/build-arch.sh \
  scripts/install-arch.sh scripts/uninstall-arch.sh
./scripts/build-arch.sh x64
```

For ARM64:

```bash
./scripts/build-arch.sh arm64
```

Outputs:

- `dist/dns-filter-arch-x64.tar.gz`
- `dist/dns-filter-arch-arm64.tar.gz`

### Install

```bash
cd dist
tar -xzf dns-filter-arch-x64.tar.gz
cd dns-filter-linux-x64
sudo ./install-arch.sh
```

The installer:

- verifies Arch, `pacman`, and `systemd`;
- installs required runtime libraries;
- backs up an existing configuration and database;
- installs under `/opt/dns-filter`;
- installs and enables `dns-filter.service`;
- waits for `/healthz` before changing DNS;
- configures DNS through `systemd-resolved`, NetworkManager, or a backed-up
  `/etc/resolv.conf` fallback;
- installs an application-menu entry and diagnostic tools.

Install without changing system DNS:

```bash
sudo ./install-arch.sh --no-dns
```

Skip dependency installation only when the required packages are already present:

```bash
sudo ./install-arch.sh --skip-deps
```

### Open and verify

```bash
/opt/dns-filter/desktop/DNSFilter
sudo /opt/dns-filter/arch-diagnostics.sh
curl http://127.0.0.1:8080/healthz
```

The dashboard is available at `http://127.0.0.1:8080`.

Useful service commands:

```bash
systemctl status dns-filter.service
sudo systemctl restart dns-filter.service
journalctl -u dns-filter.service -f
```

### Native Arch security controls

Connection Freeze uses a private `inet dnsfilter_freeze` nftables table. It allows
loopback, DNS, DHCP, ICMP, and established flows while blocking new outbound flows.
Stopping the freeze or service removes that private table.

App Network Control groups matching executable processes with cgroup v2 and blocks
their outbound traffic through the private `inet dnsfilter_apps` table. It records
the original cgroup before moving a process and restores it during Allow. A watcher
adds newly launched matching processes while the rule remains enabled.

Requirements:

- The Go service runs as root through systemd.
- Unified cgroup v2 is mounted at `/sys/fs/cgroup`.
- `nftables` is installed and supported by the kernel.

Test per-application control with a non-critical graphical application. Never
select a terminal, desktop shell, login manager, systemd, or a process required to
administer the computer.

Follow [docs/ARCH-TEST-CHECKLIST.md](docs/ARCH-TEST-CHECKLIST.md) during the first
real-machine validation.

### Update

Extract a newer Arch archive and run `sudo ./install-arch.sh` again. The installer
preserves active configuration and data and creates a timestamped safety backup in
`/var/lib/dns-filter-backups` before replacing program assets.

### Uninstall

```bash
sudo ./uninstall-arch.sh
```

Normal uninstall releases DNS Filter-owned app-network rules, restores automatic
DNS, disables the service, and preserves configuration/data under
`/var/lib/dns-filter-preserved`.

Permanent removal:

```bash
sudo ./uninstall-arch.sh --purge
```

`--purge` also removes preserved data and installer backups.

## Recovery commands

Windows commands require Administrator PowerShell. Arch commands require `sudo`.

| Purpose | Windows | Arch Linux |
|---|---|---|
| Health | `Invoke-RestMethod http://127.0.0.1:8080/healthz` | `curl http://127.0.0.1:8080/healthz` |
| Restore automatic DNS | `.\dns-filter.exe reset-dns` | `/opt/dns-filter/dns-filter reset-dns` |
| Release per-app rules | `.\dns-filter.exe reset-app-network` | `/opt/dns-filter/dns-filter reset-app-network` |
| Service logs | Windows launcher/log files | `journalctl -u dns-filter.service -n 100 --no-pager` |
| Full diagnostics | Dashboard/System Info | `/opt/dns-filter/arch-diagnostics.sh` |

If an Arch network-control recovery command cannot execute, deleting only the two
private nftables tables restores traffic without touching other firewall tables:

```bash
sudo nft delete table inet dnsfilter_apps
sudo nft delete table inet dnsfilter_freeze
```

## Installed Arch paths

| Path | Purpose |
|---|---|
| `/opt/dns-filter/dns-filter` | Go service and recovery commands |
| `/opt/dns-filter/desktop/DNSFilter` | Avalonia desktop client |
| `/opt/dns-filter/configs/` | Active configuration and custom rules |
| `/opt/dns-filter/data/` | SQLite database, logs, Vault, and local state |
| `/etc/systemd/system/dns-filter.service` | Background service |
| `/usr/share/applications/dns-filter.desktop` | Application-menu entry |
| `/var/lib/dns-filter-backups/` | Timestamped update backups |
| `/var/lib/dns-filter-preserved/` | State retained by normal uninstall |

Do not publish active configuration or data directories. They can contain password
hashes, recovery/Vault state, plugin settings, and private local activity history.
