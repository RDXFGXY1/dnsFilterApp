# Release Checklist

This checklist is the gate for the next public package after `v0.1.0`.
Checks recorded for the old release do not certify the `v0.2.0` draft. The
source-only checks below must be repeated against the packaged build before
the draft is published.

## Automated verification

- [ ] Go formatting passes.
- [ ] Go tests pass on Linux and Windows using the version declared in `go.mod`.
- [ ] Every Go command builds on Linux and Windows.
- [ ] The Avalonia desktop restores and builds in Release configuration for Windows x64 and Arch x64.
- [ ] Dashboard JavaScript passes syntax validation.
- [ ] The Hard System extension JavaScript passes syntax validation and the unpacked extension loads without errors.

## Clean Windows installation

- [ ] First-run setup creates the administrator and Secure Vault.
- [ ] Both one-time recovery credentials can be saved and independently verified.
- [ ] DNS listener, firewall, elevation, and System DNS checks report accurately.
- [ ] Login, logout, Vault lock/unlock, account recovery, and Vault recovery work.
- [ ] Configure custom SMTP and verify a fresh Vault recovery code arrives and
      expires correctly; verify offline USB recovery separately.
- [ ] Verify Vault time/request unlock limits and manual lock.
- [ ] Install an `.adnf` Side App, launch it from the navbar, then disable and
      remove it without changing the core dashboard.
- [ ] Verify local dashboard hostname, sign-in after a domain change, and exact
      LAN IP allowlisting; public-Internet access remains unavailable.
- [ ] Verify Live history persists across reload/restart, retains at most 200
      entries, and respects privacy settings; check top blocked domains and the
      DNS health timeline.
- [ ] Run Quick, Full, data-limited, and timed Network Tests; confirm data use,
      cancellation, and saved 30-result history.
- [ ] Verify Discipline Needs Attention filters and rescue suggestions.
- [ ] Verify tray icon and deadline notifications in a signed-in session.
- [ ] Test before-sign-in startup and DNS fallback after a real Windows reboot.
- [ ] Filtering survives restart, sleep/resume, and network-adapter changes.
- [ ] A Focus Capsule can be created, edited, selected, recovered after restart, and ended safely.
- [ ] Focus Capsule website and YouTube channel decisions work after reloading the browser extension.
- [ ] A Focus Capsule app allowlist blocks only eligible non-allowed applications and releases every temporary rule when the session ends.

## Clean Arch installation

- [ ] The x64 archive installs through `install-arch.sh` on a clean Arch or Arch-based machine.
- [ ] `dns-filter.service` starts automatically after reboot and `/healthz` reports the packaged version.
- [ ] DNS setup and reset work without damaging the original resolver configuration.
- [ ] Connection Freeze passes the real-machine checks in `installation/arch/TEST-CHECKLIST.md`.
- [ ] App Network Control and Focus Capsule allowlists pass a real-machine cgroup v2 and nftables test.
- [ ] `arch-diagnostics.sh` reports enough information to recover a failed installation without exposing secrets.

## Upgrade and removal

- [ ] Upgrade preserves configuration, database, Vault, and recovery state.
- [ ] Failed upgrade rolls back to the previous executable.
- [ ] Uninstall removes owned firewall rules and restores automatic System DNS.
- [ ] User data removal is a separate, explicit choice.

## Release artifacts

- [ ] Version matches `internal/version/VERSION` everywhere.
- [ ] Runtime secrets, databases, logs, certificates, and Vault files are absent
      from the new package.
- [ ] `scripts/audit-release.ps1` passes for both the new portable folder and ZIP.
- [ ] Windows executable and installer are Authenticode-signed.
- [ ] SHA-256 checksum and machine-readable manifest are generated and verified
      for the new release ZIP.
- [ ] Release notes list behavior changes, limitations, and recovery instructions.
