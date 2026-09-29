# App Network Control

App Network Control blocks outbound internet access for one Windows executable while the rest of the computer remains online. It is separate from DNS application policies: DNS policies block known service domains, while App Network Control creates a Windows Firewall rule for the selected `.exe` path.

## Use it

1. Open **Protection → Applications**.
2. Find a running application, or paste the full path to an `.exe` file.
3. Select **Block**. Unlock the Configuration Vault when requested.
4. Select **Allow** to remove the managed firewall rule.

The panel shows running applications, current managed blocks, and the latest control changes. Changes require DNS Filter to be running as Administrator.

## Behavior and safety

- Version 1 controls outbound connections for all Windows network profiles and covers IPv4 and IPv6 through Windows Firewall.
- Existing connections may remain alive briefly. Restart the selected application if it does not disconnect immediately.
- DNS Filter refuses to block its own executable and a small set of critical Windows processes.
- A rule is tied to the full executable path. If an application updates into a new folder, select its new path.
- The activity list records rule changes, not every blocked packet. Deep per-packet telemetry requires a future Windows Filtering Platform component.
