# Connection Freeze (Windows)

Open **Connection Freeze** in the dashboard sidebar, or click **Connection Freeze
& logs** in the Go desktop window. Sign in, select a duration, and press **Freeze
new connections**. Start requires the configuration vault to be unlocked if it is
enabled. **Stop freeze** remains available to signed-in users even when the vault
is locked.

The server must run as Administrator. The feature blocks new outbound connections
on IPv4 and IPv6, including LAN destinations, through Windows Filtering Platform.
Existing Windows-authorized flows remain eligible to continue; other firewall rules
still apply. When an app reconnects, switches servers or starts another connection,
that new connection is blocked. An already connected game can keep working, and
offline games are unaffected.

Exceptions: loopback connections, the DNS Filter server executable (for DNS and
its own maintenance traffic), and DHCP renewal traffic. Local proxies and existing
VPN tunnels can carry traffic within an existing allowed connection; this feature
is a focus aid, not an isolation boundary against an administrator or proxy bypass.
It does not block new inbound connections or replace the existing firewall.

## Duration and recovery

Choose **Until I stop it manually** or 1–120 minutes (the interface offers 15, 30,
60 and 120). Manual mode uses `minutes: 0` and has no expiry timer; the API requires
an explicit duration field. Rules belong to a
dynamic WFP session: stopping the freeze, timer expiry, server exit, crash or reboot
removes them. The feature starts **off** after restart. Closing only the desktop
window does not stop the server or its active freeze.

## Connection logs

The log shows session start/stop, Windows application path, destination IP and port,
protocol, timestamp and reason for blocked new connections. These are WFP events,
not guessed DNS-to-process associations. Windows may omit/coalesce repeated
attempts; logs are best effort and do not contain traffic content. Log failures
appear in the status area. Up to 1,000 entries are retained in the local database;
the most recent 200 are displayed and searchable. Logs survive a restart.

## Implementation and verification

Rules at ALE_AUTH_CONNECT_V4/V6 match initial, non-loopback authorization and leave
reauthorizations of existing flows alone. No persistent Defender Firewall profile
settings are changed. Partial setup errors close the dynamic session.

Live Windows validation checks a blocked new external TCP connection, survival of
an established TLS connection, loopback access, the matching application/destination
event, and restored connectivity after stopping. The opt-in integration test needs
Administrator permission and DNS_FILTER_WFP_TEST=1; always run with a 20-second
test timeout. Normal unit tests never change the firewall.
