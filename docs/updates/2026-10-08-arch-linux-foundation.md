# Arch Linux foundation and native controls

Release date: 2026-10-08

## Added

- Dedicated x64 and ARM64 Arch/Arch-based build archives.
- Arch installer and uninstaller with `pacman` dependency handling and systemd
  service registration.
- Service health checker and a read-only Arch diagnostics command.
- Linux automatic-startup management through `systemctl`.
- Connection Freeze through the private `dnsfilter_freeze` nftables table.
- Per-application network control through cgroup v2 and the private
  `dnsfilter_apps` nftables table.
- `reset-app-network` recovery command for block, quarantine, and schedule rules.
- Real-machine validation checklist with emergency network recovery commands.

## Changed

- The dashboard's App Network Control panel now identifies Windows or Linux at
  runtime and accepts a platform-native executable path.
- App rule intent is restored from SQLite after service restart.
- Arch installation waits for a healthy API before routing system DNS.
- The systemd unit now includes restart policy, timeouts, owner-only file mask,
  journald output, and Connection Freeze cleanup.
- Official Linux scope is now Arch and Arch-based distributions; generic Linux
  and macOS are not claimed as supported targets.

## Fixed

- Arch updates preserve configuration, database, Vault, rules, and logs and create
  a timestamped safety backup.
- Data preserved by normal uninstall is restored during a later clean install.
- An application can be allowed after its executable has been removed or moved.
- Failed restoration no longer leaves a rule displayed as protected when its
  operating-system control could not be applied.
- Normal Arch uninstall releases application network rules and restores automatic
  DNS before removing the service.

## Security notes

- Both Linux network features use dedicated nftables tables and do not replace
  distribution or user firewall tables.
- Per-application control records original process cgroups before moving them and
  restores those memberships during Allow/recovery.
- Native Linux controls require root, nftables, and unified cgroup v2.
- The backend is build-tested but must still be validated on real Arch hardware
  before production use.

## Verification

- Focused Go tests passed for API, App Network Control, and Connection Freeze.
- Linux-specific packages compiled into test binaries for `linux/amd64`.
- The complete Go service cross-compiled successfully for `linux/amd64`.
- App Network Control JavaScript passed syntax validation.

## Related application work in this development cycle

- Discipline gained a full Task Hub, Daily Journal, Focus Activity Logger, and
  Intelligence Inbox.
- The Intelligence Inbox supports local deterministic briefings plus explicit
  AI-assisted analysis through Vault-encrypted OpenAI, xAI/Grok, Gemini, or
  compatible provider credentials.
- The plugin platform gained managed runtimes, native API-generated controls,
  sandboxed local web views, and the expanded TinyPrinter Studio integration.
