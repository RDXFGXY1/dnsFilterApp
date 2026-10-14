# Install DNS Filter on Arch Linux

## Status

Arch Linux and Arch-based distributions using `pacman` and `systemd` are the
official Linux target. Real-machine testing has verified the DNS service,
dashboard, Avalonia desktop, and automatic startup after reboot.

The latest Connection Freeze correction and App Network Control still require
the checks in [TEST-CHECKLIST.md](TEST-CHECKLIST.md) before they are marked fully
verified.

## Requirements for building from source

- Go matching the version declared in `go.mod`.
- .NET 8 SDK.
- `bash`, `tar`, and standard build tools.

The installer normally installs the required .NET runtime, Avalonia libraries,
`curl`, and `nftables` through `pacman`.

## Build

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

## Install

```bash
cd dist
tar -xzf dns-filter-arch-x64.tar.gz
cd dns-filter-linux-x64
sudo ./install-arch.sh
```

The installer verifies the platform, installs runtime dependencies, backs up an
existing installation, installs to `/opt/dns-filter`, enables the systemd
service, waits for a healthy API, and then configures system DNS.

Optional modes:

```bash
# Install without changing system DNS
sudo ./install-arch.sh --no-dns

# Do not install dependencies that are already present
sudo ./install-arch.sh --skip-deps
```

## Open and verify

```bash
/opt/dns-filter/desktop/DNSFilter
curl http://127.0.0.1:8080/healthz
sudo /opt/dns-filter/arch-diagnostics.sh
```

Dashboard: `http://127.0.0.1:8080`

```bash
systemctl status dns-filter.service
sudo systemctl restart dns-filter.service
journalctl -u dns-filter.service -f
```

## Native network controls

Connection Freeze uses the private `inet dnsfilter_freeze` nftables table. It
keeps established flows and essential local/DNS/DHCP traffic available while
dropping other outbound traffic until the freeze stops.

App Network Control uses cgroup v2 and the private `inet dnsfilter_apps` table to
isolate matching executable processes. The original cgroup is recorded and
restored when the application is allowed again.

Both controls require the root systemd service, nftables, and—App Network Control
specifically—unified cgroup v2. Test them only with a non-critical graphical
application. Never select the terminal, desktop shell, login manager, systemd, or
another process required to recover the computer.

## Update

Extract a newer archive and run `sudo ./install-arch.sh` again. Active state is
preserved and a timestamped backup is created under
`/var/lib/dns-filter-backups`.

## Uninstall

```bash
sudo ./uninstall-arch.sh
```

Normal uninstall restores automatic DNS, releases DNS Filter-owned network
controls, disables the service, and preserves state under
`/var/lib/dns-filter-preserved`.

Permanent removal:

```bash
sudo ./uninstall-arch.sh --purge
```

## Emergency recovery

```bash
# Works even when the DNS Filter binary or API is unavailable
sudo /opt/dns-filter/emergency-restore-dns.sh
# Re-enable later with: sudo systemctl enable --now dns-filter.service

# Normal application-assisted recovery
sudo /opt/dns-filter/dns-filter reset-app-network
sudo /opt/dns-filter/dns-filter reset-dns

# Last-resort removal of DNS Filter's private tables only
sudo nft delete table inet dnsfilter_apps
sudo nft delete table inet dnsfilter_freeze
```

## Installed paths

| Path | Purpose |
|---|---|
| `/opt/dns-filter/dns-filter` | Go service and recovery commands |
| `/opt/dns-filter/desktop/DNSFilter` | Avalonia desktop client |
| `/opt/dns-filter/configs/` | Configuration and custom rules |
| `/opt/dns-filter/data/` | Database, logs, Vault, and local state |
| `/opt/dns-filter/emergency-restore-dns.sh` | Standalone automatic-DNS recovery |
| `/etc/systemd/system/dns-filter.service` | Background service |
| `/usr/share/applications/dns-filter.desktop` | Application-menu entry |
| `/var/lib/dns-filter-backups/` | Update backups |
| `/var/lib/dns-filter-preserved/` | State retained by normal uninstall |

Never publish active `configs` or `data` directories. They can contain password
hashes, Vault/recovery state, plugin settings, and private activity history.
