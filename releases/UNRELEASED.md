# UNRELEASED

This file is a running list of changes that have not yet been shipped in a formal release.

When preparing a release, move relevant entries from here into a versioned file (for example `v1.0.0.md`).

Format
- **Added**: New features or public API additions
- **Changed**: Behaviour changes, improvements
- **Fixed**: Bug fixes
- **Removed**: Removed features or breaking changes
- **Security**: Any security-related fixes

--

## Added

- Native Windows WPF control center with dark-only monitoring and controls.
- First-run administrator and Secure Vault onboarding with separate one-time recovery credentials.
- First-run checks for the DNS listener, Administrator rights, Windows Firewall, and System DNS routing.
- Encrypted DNS, DNS-bypass monitoring, signed security packs, Download Guardian, application network control, connection intelligence, profiles, schedules, and emergency controls.
- Canonical application version shared by the Go service, Windows desktop, dashboard, and packaging scripts.

## Changed

- Runtime configuration, Vault data, databases, logs, and machine approval state are excluded from release source packages.
- Fresh packages receive a sanitized configuration template instead of the developer's local configuration.
- Continuous integration verifies Go on Linux and Windows, builds the WPF desktop, and checks JavaScript syntax.
- Windows portable releases include a health-checked installer/updater with automatic executable and web-asset rollback.
- Windows releases include a scoped uninstaller that restores automatic DNS and removes only DNS Filter-owned firewall, startup, service, and certificate resources.

## Fixed

- Existing installations retain their account and Vault state when upgrading.
- Windows updates preserve configuration, database, Vault, logs, certificates, and recovery state instead of replacing mutable folders.
- Windows uninstall preserves configuration and local data by default; destructive data removal requires an explicit confirmation phrase.
- System DNS routing intent is persisted after explicit activation and DNS Route Watch repairs adapter changes or resume-time route loss automatically.
- Setup closes permanently after successful onboarding and is restricted to direct loopback requests.
- The Vault auto-lock choice is applied immediately during first-run setup.
- Configuration writes are atomic and scoped to the Config instance that loaded the file.
- Failed module, block-action, password, and blocklist-setting writes now roll back runtime state instead of reporting false success.
- Encrypted Vault restore retains the current installation's configuration path.
- Account recovery enforces the full password policy, rotates the recovery key, and invalidates every existing session.
- Vault password and recovery failures share a temporary lockout, and unavailable Vault endpoints now return a clear service error.
- The Windows desktop shows legacy one-time account recovery keys in a mandatory save dialog instead of discarding them after login.

## Removed

- Removed the broken static GitHub Pages deployment; the dashboard requires the local Go service.

## Security

- Removed tracked runtime secrets and state from release inputs.
- Replaced shared default-login behavior with strong first-run credential creation.
- Configuration files are saved with owner-only permissions on supported systems.
- Setup sessions use cryptographically random, server-side, HttpOnly, SameSite=Strict cookies.
- Added DNS-rebinding protection that accepts only the configured dashboard host and trusted loopback names.
- Added per-session CSRF protection to every authenticated API mutation in both the web and Windows interfaces.
- Added browser hardening headers for framing, MIME sniffing, referrers, permissions, and content sources.
- Added per-account and per-client sign-in throttling with a five-attempt temporary lockout.
- Added HTTP read, write, idle, header, and request-body limits to reduce slow-client and oversized-request attacks.
- Replaced broad inline-content permission with a unique per-response CSP nonce, default-deny resource policy, and narrowly scoped legacy event/style-attribute compatibility.
- Added a mandatory portable-release audit that rejects runtime state, private material, credential patterns, debug symbols, developer paths, and unsafe archive entries without printing secret values.
- Added an Authenticode release gate that signs and verifies first-party Windows binaries and maintenance scripts before the portable ZIP is created.
- Added verified SHA-256 checksum and machine-readable manifest generation for every portable release archive.

Notes
- Keep entries small and include links to issues/PR numbers when available.
