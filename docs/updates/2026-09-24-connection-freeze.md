# Connection Freeze

- Added a dedicated Connection Freeze dashboard page and a direct button in the Go desktop app.
- New outbound IPv4/IPv6 connections are blocked while established flows remain eligible.
- Added 1–120-minute expiry, manual recovery, and dynamic rule cleanup on server exit.
- Added an explicit “Until I stop it manually” duration with no expiry timer; server exit still removes the freeze safely.
- Improved control spacing, responsive layout, searchable log presentation, active-state badges, and the sidebar snowflake icon.
- Kept loopback, DNS Filter's own traffic, and DHCP available.
- Added searchable, locally retained connection events with application paths, IPs, ports and protocols.
- Kept stop/recovery accessible to authenticated users when the configuration vault is locked.

Validated with an elevated live Windows test: new external TCP blocked, established
TLS and loopback usable, blocked application event received, and connectivity restored
after stop. Unit tests cover invalid/duplicate starts, expiry, stale timer callbacks,
start failure, log persistence, authentication, and vault-locked recovery.

See [Connection Freeze](../features/connection-freeze.md) for limitations and usage.
Launch `build/DNS-Filter-Go-Freeze.exe`, then click **Connection Freeze & logs**.
