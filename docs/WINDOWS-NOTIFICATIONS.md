# Windows tray and deadline notifications

The Avalonia desktop app now has a DNS Filter icon in the Windows notification area (beside the clock, possibly under the hidden-icons arrow). It uses the same icon for its executable, window, Start menu shortcut, and Windows notifications.

## Behavior

- Clicking the tray icon opens the existing window. Closing the window hides it; the tray companion and deadline checks continue.
- The tray menu opens the desktop or Discipline, enables/disables deadline notifications, controls **Start tray with Windows**, sends a test notification, and exits the desktop companion.
- Exiting the companion does not stop the Go DNS service. Native notifications require the companion to be running in the signed-in Windows user's session.
- First Windows launch registers a current-user Start menu shortcut, application identity `DNSFilter.Desktop`, a `dnsfilter-app:` activation protocol, and an HKCU startup entry. No Administrator password, account password, reusable API credential, or elevated notification process is needed. The tray starts at Windows sign-in; the separate DNS service can start before sign-in.
- Clicking a toast opens Discipline. Reopening the desktop or activating a toast reuses the existing tray process instead of creating duplicate icons.

## Deadline sources and timing

The read-only local companion includes unfinished Discipline task deadlines, explicit task reminders, reminders, projects, goals, and milestones under active projects. Completed/archived work and paused goals/projects are excluded. Plain dashboard todos currently have no deadline field and are not treated as dated tasks. Independently installed Side Apps keep their own data and are not scanned.

Checks run every 20 seconds. A notification is eligible 15 minutes before the deadline and once more when due. Date-only deadlines are interpreted as **09:00 on that date in the PC's local time zone**. Deadlines missed within the last 24 hours are caught up, at most three notifications per check, to avoid a large flood. Older overdue work remains visible in Discipline without generating old toast alerts.

Delivery keys and preferences are persisted under `%LOCALAPPDATA%\DNSFilter\desktop-notifications.json`; restarting Windows/the companion does not replay the same delivery stage. A new due time gets a new key. Task reminder/deadline entries for the same task and exact time are deduplicated. The Windows Notification Center retains accepted toasts according to Windows settings. Do Not Disturb and Windows notification permissions control whether banners/sounds appear.

The native name/icon follow the Windows desktop [AppUserModelID and shortcut model](https://learn.microsoft.com/en-us/windows/win32/shell/enable-desktop-toast-with-appusermodelid). The provider uses the built-in Windows PowerShell 5.1 WinRT bridge. Toast XML is constructed as XML, not executable PowerShell text; task names are escaped, including Unicode and markup characters. No third-party notification service is contacted.

## Data and packaging

`cmd/desktop-notifications` reads the existing SQLite database in read-only mode. It never creates/migrates tables, marks work completed, or modifies DNS/firewall settings. Keeping it local avoids granting unauthenticated notification API access or saving login passwords. The configuration vault does not block this personal-data reader.

Build the companion before the desktop:

```powershell
go build -o build\dns-filter-notifications.exe ./cmd/desktop-notifications
dotnet build application\DnsFilter.Avalonia\DnsFilter.Avalonia.csproj -c Release
```

The Windows portable builder includes the helper automatically. Keep `DNSFilter.exe`, its assemblies, `Assets/`, `Notifications/`, and `dns-filter-notifications.exe` together. In source builds the desktop locates the repository configuration through its parent directories. The icon SVG source is at `application/DnsFilter.Avalonia/Assets/dns-filter.svg`.

## Validation

Go tests cover due/soon stages, repeat keys, reminder/deadline deduplication, completed/archived/old work, all supported sources, local/date-time parsing, read-only behavior, and paths containing spaces or `#`. The C# checks cover preferences/delivery persistence, pruning, Unicode/XML escaping, and the branded image. Release builds pass with zero warnings/errors.

A live Windows test confirmed notification permission `Enabled` and found the branded test toast in `DNSFilter.Desktop` notification history. The tray companion remains running in the user's session. A real sign-out/reboot startup test has not been performed.
