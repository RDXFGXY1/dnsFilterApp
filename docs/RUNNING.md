# Running DNS Filter on Windows, Linux & macOS

This guide walks through building, running, and pointing your system at the DNS
filter on each OS — including the two gotchas that silently bypass filtering
(**browser Secure DNS** and **IPv6 DNS**).

---

## Testing status

| Platform | Build | Runtime tested | Notes |
|----------|:-----:|:--------------:|-------|
| **Windows 11** | ✅ | ✅ Fully tested | End-to-end: build, run, `setup-dns`, blocking, dashboard. IPv6 DNS must be handled (see below). |
| **Linux** | ✅ | ⚠️ Not yet tested | Cross-compiles cleanly. `setup-dns` uses `resolvectl` / `/etc/resolv.conf`. |
| **macOS** | ✅ | ⚠️ Not yet tested | Cross-compiles cleanly. `setup-dns` uses `networksetup`. |

The binary is **pure Go** (uses `modernc.org/sqlite`), so it builds with
`CGO_ENABLED=0` and needs **no C compiler** on any platform.

---

## 0. Prerequisites

- **Go 1.21+** to build (not needed if you already have a prebuilt binary)
- **Administrator / root** to run (binds DNS port 53 and block-page port 80)

---

## 1. Build

The same commands work everywhere. Set `CGO_ENABLED=0` to get a fully static,
portable binary.

### Windows (PowerShell)
```powershell
$env:CGO_ENABLED=0
go build -o build/dns-filter.exe ./cmd/server
```

### Linux / macOS (bash)
```bash
CGO_ENABLED=0 go build -o build/dns-filter ./cmd/server
```

### Cross-compile for another OS
```bash
# From any machine — build for Linux:
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o build/dns-filter-linux   ./cmd/server
# ...for Windows:
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/dns-filter.exe      ./cmd/server
# ...for macOS (Intel amd64 / Apple Silicon arm64):
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o build/dns-filter-darwin   ./cmd/server
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -o build/dns-filter-darwin-arm64 ./cmd/server
```

---

## 2. Run the filter

Keep this process running the whole time you want filtering active.

### Windows (Administrator PowerShell)
```powershell
.\build\dns-filter.exe
```

### Linux / macOS
```bash
sudo ./build/dns-filter
```

You should see the banner, then (in `data/logs/dns-filter.log`):
`DNS server listening on 0.0.0.0:53` and `Web Dashboard: http://127.0.0.1:8080`.

> If it prints the banner and exits immediately, another instance is already
> using ports 53/80/8080. Check `data/logs/dns-filter.log` for a `bind` error and
> stop the old instance.

---

## 3. Point your system DNS at the filter

The binary can do this for you (auto-detects your active network interface):

### Windows (Administrator)
```powershell
.\build\dns-filter.exe setup-dns
```

### Linux / macOS
```bash
sudo ./build/dns-filter setup-dns
```

To undo it later (**always do this when you stop the filter**, or DNS stops working):
```bash
# Windows:  .\build\dns-filter.exe reset-dns
# Linux/mac: sudo ./build/dns-filter reset-dns
```

You can also toggle this from the dashboard's **Middleman** button (the server
must be running elevated for the button to work).

---

## 4. ⚠️ Close the two bypass routes

Filtering happens at the DNS layer. Two things route DNS *around* the filter and
must be disabled, or blocked sites will still load.

### 4a. Browser Secure DNS (DNS-over-HTTPS)
Modern browsers resolve domains over their own encrypted DNS by default, ignoring
your system DNS. Turn it off:

- **Brave:**  `brave://settings/security` → **Use secure DNS** → **Off**
- **Chrome:** `chrome://settings/security` → **Use secure DNS** → **Off**
- **Edge:**   `edge://settings/privacy` → **Use secure DNS** → **Off**
- **Firefox:** `about:preferences#privacy` → scroll to **DNS over HTTPS** → **Off**

Then fully quit and reopen the browser (they cache DNS internally).

### 4b. IPv6 DNS (the sneaky one)
If your router advertises an IPv6 DNS server, Windows/Linux/macOS will prefer it
and skip the IPv4 filter at `127.0.0.1`. Symptom: `nslookup xnxx.com` shows
`Server: fe80::1` and returns real IPs.

**Windows** — disable IPv6 on the active adapter (replace `"Wi-Fi 2"` with yours;
find it via `Get-NetAdapter`):
```powershell
Disable-NetAdapterBinding -Name "Wi-Fi 2" -ComponentID ms_tcpip6
ipconfig /flushdns
# Undo later:
Enable-NetAdapterBinding -Name "Wi-Fi 2" -ComponentID ms_tcpip6
```

**Linux** — disable IPv6 temporarily:
```bash
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1
sudo sysctl -w net.ipv6.conf.default.disable_ipv6=1
# Undo: set both back to 0
```

**macOS** — set IPv6 to Link-local only (replace `Wi-Fi` with your service):
```bash
sudo networksetup -setv6off "Wi-Fi"
# Undo: sudo networksetup -setv6automatic "Wi-Fi"
```

> A permanent fix (filter listening on IPv6 + `setup-dns` handling IPv6) is on the
> roadmap so this manual step won't be needed.

---

## 5. Flush the OS DNS cache

```bash
# Windows:
ipconfig /flushdns
# Linux (systemd):
sudo resolvectl flush-caches
# macOS:
sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder
```

---

## 6. Test that blocking works

```bash
# Should return 127.0.0.1  (blocked):
nslookup xnxx.com
# Should return a real public IP (allowed):
nslookup google.com
```

If `xnxx.com` resolves to `127.0.0.1`, filtering is live. Open the site in your
browser to confirm it no longer loads.

---

## 7. Open the dashboard

Browse to **http://127.0.0.1:8080** and log in:

- Username: `admin`
- Password: `changeme`  *(change this in `configs/config.yaml`)*

From here you can manage categories, keyword lists, whitelist, custom blocks,
gamification stats, and toggle Middleman mode.

---

## Quick reference

| Action | Windows | Linux / macOS |
|--------|---------|---------------|
| Build | `go build -o build/dns-filter.exe ./cmd/server` | `go build -o build/dns-filter ./cmd/server` |
| Run | `.\build\dns-filter.exe` | `sudo ./build/dns-filter` |
| Point DNS → filter | `.\build\dns-filter.exe setup-dns` | `sudo ./build/dns-filter setup-dns` |
| Restore DNS | `.\build\dns-filter.exe reset-dns` | `sudo ./build/dns-filter reset-dns` |
| Flush DNS cache | `ipconfig /flushdns` | `resolvectl flush-caches` (Linux) |
| Test block | `nslookup xnxx.com` | `nslookup xnxx.com` |
| Dashboard | http://127.0.0.1:8080 | http://127.0.0.1:8080 |

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Banner prints then exits | Ports 53/80/8080 already in use | Stop the old instance (check the log for `bind` error) |
| Blocked site still loads | Browser Secure DNS (DoH) | Disable it (§4a), restart browser |
| `nslookup` shows `Server: fe80::1` | IPv6 DNS bypassing filter | Disable IPv6 DNS (§4b) |
| `setup-dns` fails | Not elevated | Run as Administrator / `sudo` |
| Internet stops after closing filter | System DNS still points at 127.0.0.1 | Run `reset-dns` |
