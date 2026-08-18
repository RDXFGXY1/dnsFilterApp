# DNS Filter - Background Service Scripts

These scripts allow you to run DNS Filter in the background automatically on Windows.

## Files

| File | Purpose |
|------|---------|
| `start-dns-filter.bat` | **Main launcher** - Double-click to start DNS Filter |
| `start-dns-filter.ps1` | PowerShell script that handles startup (called by .bat) |
| `stop-dns-filter.bat` | **Stopper** - Double-click to stop DNS Filter |
| `stop-dns-filter.ps1` | PowerShell script that handles shutdown (called by .bat) |

---

## Quick Start

### First Time Setup

1. **Right-click `start-dns-filter.bat`**
2. **Select "Run as Administrator"**
3. **Approve the dialog** (appears once)
4. DNS Filter will start in the background
5. System DNS will be automatically configured

### Subsequent Runs

Just double-click `start-dns-filter.bat` - no approval needed again!

---

## What Each Script Does

### `start-dns-filter.bat` (The Easy Way)
✅ Checks for Administrator privileges
✅ Launches the PowerShell startup script
✅ Runs hidden (no visible window)

### `start-dns-filter.ps1` (The Real Work)
✅ **First Run:**
  - Shows approval dialog
  - Saves approval flag (in `data/.dns-filter-approved`)
  - Requires explicit user consent

✅ **On Subsequent Runs:**
  - Checks if DNS Filter already running
  - Starts DNS Filter hidden in background
  - Automatically runs `setup-dns` to configure system DNS
  - Logs all activity to `data/logs/dns-filter-startup.log`

✅ **Checks:**
  - Admin privileges (exits if not elevated)
  - DNS Filter executable exists
  - Process not already running

### `stop-dns-filter.bat` & `stop-dns-filter.ps1`
✅ Stops the running DNS Filter process
✅ Restores original system DNS settings
✅ Flushes DNS cache
✅ All with admin check and error handling

---

## How to Use

### Start DNS Filter (Hidden in Background)
```
Double-click → start-dns-filter.bat
```

### Stop DNS Filter
```
Double-click → stop-dns-filter.bat
```

### Check Status
```powershell
# In PowerShell:
Get-Process dns-filter

# If running, you'll see the process details
# If not running, you'll get "cannot find" error
```

### View Logs
```
Open: data/logs/dns-filter-startup.log
```

---

## First-Run Approval Process

### What Happens:
1. Script checks if `data/.dns-filter-approved` exists
2. **If NOT found** (first run):
   - Shows a dialog asking for approval
   - **Click "Yes"** → Approval recorded, script exits
   - **Click "No"** → Exits without approval
3. **If found** (subsequent runs):
   - Skips approval dialog and starts immediately

### To Reset Approval (Start Over):
```powershell
# In PowerShell (as Admin):
Remove-Item "data\.dns-filter-approved"
```

Next time you run `start-dns-filter.bat`, you'll see the approval dialog again.

---

## Behind the Scenes (What's Actually Happening)

### On Startup:
```
1. Administrator check
2. Check if already running
3. Create data/logs directories if needed
4. Start dns-filter.exe hidden (CreateNoWindow)
5. Wait 2 seconds
6. Verify process is still running
7. Run setup-dns (configure 127.0.0.1 as system DNS)
8. Log all activity
```

### On Shutdown:
```
1. Administrator check
2. Stop dns-filter process (if running)
3. Run reset-dns (restore original DNS)
4. Flush DNS cache (ipconfig /flushdns)
5. Log all activity
```

---

## Troubleshooting

### "This script requires Administrator privileges"
❌ You didn't run as Administrator
✅ **Fix:** Right-click the .bat file → "Run as Administrator"

### "DNS Filter executable not found"
❌ The binary wasn't built
✅ **Fix:** Run in PowerShell:
```powershell
go build -o build\dns-filter.exe .\cmd\server
```

### DNS Filter starts but immediately stops
❌ Another process is using ports 53, 80, or 8080
✅ **Fix:** Check `data/logs/dns-filter.log` for port binding errors
✅ **Fix:** Stop other DNS servers or VPNs

### DNS Filter started but still can't access blocked sites after unblocking
❌ Browser DNS cache
✅ **Fix:** 
   1. Disable "Secure DNS" in your browser (Settings → Security)
   2. Close browser completely and reopen
   3. Run: `ipconfig /flushdns`

### Can't connect to dashboard (http://127.0.0.1:8080)
❌ Firewall blocking
✅ **Fix:** Windows Firewall → Allow PowerShell through
✅ **Fix:** Check if DNS Filter is actually running:
```powershell
Get-Process dns-filter
```

---

## Advanced: Running with Parameters

### Skip first-run approval (always start immediately):
```powershell
# In PowerShell (as Admin):
.\start-dns-filter.ps1 -Force
```

### View PowerShell script details:
```powershell
Get-Content .\start-dns-filter.ps1 | more
```

---

## Security Notes

✅ Scripts check for Administrator privileges
✅ Approval flag stored locally (not sent anywhere)
✅ Logs only contain timestamps and operation status
✅ PowerShell execution policy bypassed only for this specific script
✅ All operations logged to `data/logs/dns-filter-startup.log`

---

## Dashboard Access

Once DNS Filter is running:

**URL:** http://127.0.0.1:8080
**Username:** admin
**Password:** changeme (change in `configs/config.yaml`)

---

## Important Browser Settings

After DNS Filter starts, you MUST disable Secure DNS in your browser, otherwise it will bypass the filter:

**Chrome/Edge/Brave:**
- Settings → Privacy and security → Security
- Uncheck "Use secure DNS"

**Firefox:**
- about:preferences → Privacy
- DNS over HTTPS → Off

**Safari (macOS only):**
- Not affected (uses system DNS)

---

## Quick Reference

| Action | Command |
|--------|---------|
| Start DNS Filter | Double-click `start-dns-filter.bat` |
| Stop DNS Filter | Double-click `stop-dns-filter.bat` |
| Check if running | `Get-Process dns-filter` (PowerShell) |
| View logs | `data/logs/dns-filter-startup.log` |
| Reset approval | `Remove-Item "data\.dns-filter-approved"` |
| Flush DNS cache | `ipconfig /flushdns` |
| Access dashboard | http://127.0.0.1:8080 |

---

## Need Help?

Check these files in order:
1. **Script logs:** `data/logs/dns-filter-startup.log`
2. **DNS Filter logs:** `data/logs/dns-filter.log`
3. **Documentation:** `docs/RUNNING.md`
4. **Architecture:** `docs/ARCHITECTURE.md`

---

**Created:** 2026-08-18
**Version:** DNS Filter v2.6+
