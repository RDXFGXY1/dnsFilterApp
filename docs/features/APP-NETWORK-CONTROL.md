# App Network Control

App Network Control blocks outbound network access for one executable while the
rest of the computer remains online. It is separate from DNS application policies:
DNS policy blocks known service domains, while this feature controls the operating
system process itself.

## Use it

1. Open **Protection → Applications**.
2. Select a discovered running application or paste its full executable path.
3. Press **Block** and unlock the Configuration Vault when requested.
4. Press **Allow** to restore access.

The panel shows discovered/running applications, persisted blocks, and recent rule
changes. Deleted or moved executables can still be allowed because the application
uses the stored rule identity rather than requiring the old file to exist.

## Windows backend

- Creates an outbound Windows Firewall rule for the selected `.exe` path.
- Covers IPv4 and IPv6 across all Windows network profiles.
- Requires Administrator permission.
- Rejects DNS Filter itself and critical Windows processes.

## Arch Linux backend

- Requires root, unified cgroup v2, and `nftables`.
- Creates a private cgroup for the selected executable's processes.
- Records every process's original cgroup before moving it.
- Uses only the private `inet dnsfilter_apps` nftables table.
- Watches for newly launched matching processes while the block remains enabled.
- Restores processes to their original cgroups before removing the rule.
- Reloads stored block and quarantine state after service restart.

The backend never intentionally modifies a distro/user nftables table. Test it
first with a non-critical graphical application. Never select a terminal, desktop
shell, login manager, systemd, or a process needed to recover the machine.

## Persistence and recovery

Rule intent is stored in SQLite. Temporary Linux process-origin recovery state is
stored under `/run/dns-filter`. If a stored rule cannot be restored, the manager
records a failed restoration and changes the displayed state to allowed rather than
claiming protection that is not active.

Release every DNS Filter-owned app, quarantine, and active schedule rule:

```text
dns-filter reset-app-network
```

On Arch, if the command cannot run, deleting the private table restores traffic:

```bash
sudo nft delete table inet dnsfilter_apps
```

The Arch uninstaller runs the reset command before stopping the service.

## Limitations

- Windows rule activity records management changes, not every blocked packet.
- Linux cgroup control is new and still needs validation on real Arch hardware.
- A process that opens privileged proxies or delegates traffic to another process
  may not represent a complete security boundary.
- This feature does not replace application sandboxing or malware protection.
