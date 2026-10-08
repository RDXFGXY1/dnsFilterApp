# Running DNS Filter

## Runtime model

The privileged Go service must be running before the Avalonia client can connect.
The web dashboard is served locally at `http://127.0.0.1:8080`; service readiness
is available from `/healthz`. Closing the desktop client does not stop filtering.

## Windows

Use **Install or Update DNS Filter.bat** for an installed build or run the repository
development server:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1
```

Privileged recovery commands:

```powershell
.\dns-filter.exe reset-app-network
.\dns-filter.exe reset-dns
```

## Arch Linux

```bash
systemctl status dns-filter.service
sudo systemctl restart dns-filter.service
journalctl -u dns-filter.service -f
sudo /opt/dns-filter/arch-diagnostics.sh
```

Recovery:

```bash
sudo /opt/dns-filter/dns-filter reset-app-network
sudo /opt/dns-filter/dns-filter reset-dns
```

Connection Freeze uses the private `dnsfilter_freeze` nftables table. Per-app
network control uses cgroup v2 and the private `dnsfilter_apps` table. See
[ARCH-TEST-CHECKLIST.md](ARCH-TEST-CHECKLIST.md) before the first test.

## DNS bypass checks

Browsers can bypass system DNS through Secure DNS/DoH. Disable browser Secure DNS
or point it at a resolver controlled by this application. Confirm both IPv4 and
IPv6 system DNS routes, then verify a known blocked domain appears in Live activity.

For installation and package commands, see
[INSTALL-WINDOWS-LINUX.md](../INSTALL-WINDOWS-LINUX.md).
