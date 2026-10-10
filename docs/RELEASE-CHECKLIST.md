# Release Checklist

This checklist is the minimum gate for `2.6.0-beta.1` and later builds.

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
- [x] Runtime secrets, databases, logs, certificates, and Vault files are absent.
- [x] `scripts/audit-release.ps1` passes for both the assembled portable folder and final ZIP.
- [ ] Windows executable and installer are Authenticode-signed.
- [x] SHA-256 checksum and machine-readable manifest are generated and verified for the release ZIP.
- [ ] Release notes list behavior changes, limitations, and recovery instructions.
