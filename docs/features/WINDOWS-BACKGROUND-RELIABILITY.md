# Windows Background Reliability

DNS Filter runs independently of the desktop control application. A Windows Scheduled Task launches `dns-filter.exe` as `SYSTEM` at computer startup, so protection does not wait for a user to open the GUI or sign in.

The task also has a logon trigger as a recovery path, ignores duplicate starts, runs on battery power, has no execution time limit, and retries once per minute after an unexpected non-zero exit. Normal shutdowns are not treated as crashes.

The safe Windows installer creates or refreshes this task automatically after placing the new executable. The dashboard's **Start on Login** control manages the same background task.

Listener errors are coordinated by the main process: the other listeners and data stores receive a graceful shutdown before the process returns a non-zero status for Task Scheduler to restart. Closing the HTTP server during a requested shutdown is treated as normal rather than fatal.

To repair the task manually from an Administrator PowerShell window:

```powershell
& "C:\Program Files\DNS Filter\dns-filter.exe" install-autostart
```

To remove only the startup task:

```powershell
& "C:\Program Files\DNS Filter\dns-filter.exe" uninstall-autostart
```
