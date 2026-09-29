# Release Checklist

This checklist is the minimum gate for `2.6.0-beta.1` and later builds.

## Automated verification

- [ ] Go formatting passes.
- [ ] Go tests pass on Linux and Windows using the version declared in `go.mod`.
- [ ] Every Go command builds on Linux and Windows.
- [ ] The Windows WPF desktop restores and builds in Release configuration.
- [ ] Dashboard JavaScript passes syntax validation.

## Clean Windows installation

- [ ] First-run setup creates the administrator and Secure Vault.
- [ ] Both one-time recovery credentials can be saved and independently verified.
- [ ] DNS listener, firewall, elevation, and System DNS checks report accurately.
- [ ] Login, logout, Vault lock/unlock, account recovery, and Vault recovery work.
- [ ] Filtering survives restart, sleep/resume, and network-adapter changes.

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
