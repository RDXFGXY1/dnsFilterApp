# Arch Linux validation checklist

Use this checklist on a local Arch or Arch-based desktop. Do not test Connection
Freeze over SSH because new remote connections are intentionally blocked.

## 1. Build and install

```bash
chmod +x scripts/build-arch.sh
./scripts/build-arch.sh x64
cd dist
tar -xzf dns-filter-arch-x64.tar.gz
cd dns-filter-linux-x64
sudo ./install-arch.sh
```

## 2. Verify the installation

```bash
sudo /opt/dns-filter/arch-diagnostics.sh
systemctl status dns-filter.service
curl http://127.0.0.1:8080/healthz
```

The diagnostics should report systemd, nftables, cgroup v2, the local API, and
DNS routing as available.

## 3. Test Connection Freeze

1. Keep one browser stream or download already connected.
2. Open **Connection Freeze** and choose a short duration.
3. Confirm that the existing connection remains available.
4. Try opening a new website or starting another network application; the new
   connection should fail.
5. Stop the freeze and confirm that new connections work again.

Inspect the private rules only:

```bash
sudo nft list table inet dnsfilter_freeze
```

The last rule in the `output` chain must contain both `counter drop` and the
comment `dns-filter-connection-freeze`. If the dashboard reports an error, run:

```bash
sudo /opt/dns-filter/arch-diagnostics.sh
journalctl -u dns-filter.service -n 100 --no-pager
```

## 4. Test App Network Control

1. Choose a non-critical graphical application such as Spotify, Discord, Firefox,
   or Chromium. Do not choose the terminal, desktop shell, login manager, or systemd.
2. Press **Block** and wait up to three seconds.
3. Confirm that the selected app loses network access while other applications
   remain connected.
4. Close and reopen the selected app; it should be detected and blocked again.
5. Press **Allow** and confirm that its network access returns.

Inspect the isolated rules and process cgroup:

```bash
sudo nft list table inet dnsfilter_apps
cat /proc/APP_PID/cgroup
```

Replace `APP_PID` with the selected application's process ID.

## 5. Recovery commands

Release every application rule and restore moved processes:

```bash
sudo /opt/dns-filter/dns-filter reset-app-network
```

If the command cannot run, removing the private tables immediately restores
network access. This does not touch other firewall tables:

```bash
sudo nft delete table inet dnsfilter_apps
sudo nft delete table inet dnsfilter_freeze
```

Restore automatic system DNS if required:

```bash
sudo /opt/dns-filter/dns-filter reset-dns
```

Collect diagnostics and recent logs for a bug report:

```bash
sudo /opt/dns-filter/arch-diagnostics.sh
journalctl -u dns-filter.service -n 100 --no-pager
```
