# Privacy and Security Suite

Release date: 2026-09-20

## Added

- Encrypted upstream DNS through DNS-over-HTTPS and DNS-over-TLS.
- Strict secure-DNS mode that prevents silent fallback to plaintext DNS.
- Domain decision explanations, recommendations, rule simulation, and conflict inspection.
- Emergency Lockdown with automatic expiration and a dedicated allowlist.
- Ed25519-signed security packs with trusted publisher keys and downgrade protection.
- A secure configuration vault with automatic locking, recovery codes, encrypted backups, failed-attempt lockout, and a tamper-evident audit log.
- Dashboard controls for all new interactive features.
- A unified Security Center with live feature status, readiness scoring, alerts, recommendations, and direct links to detailed controls.
- A redesigned login screen with one-time recovery-key enrollment, password reset, recovery-key rotation, attempt lockout, and random in-memory sessions.
- A complete professional interface refresh shared by the Dashboard and Discipline workspaces, using the login screen's visual language, responsive layouts, refined cards, softer controls, and unified light and dark themes.

## Changed

- DNS forwarding now uses a validated upstream pool with secure failover.
- Filtering decisions can identify a matching security pack as their source.
- DNS cache entries are cleared when an emergency or policy change could make an old answer unsafe.

## Fixed

- Corrected the syntax error in `internal/aiblock/ai_blocker.go` that prevented the package from compiling.

## Security notes

- Encrypted DNS protects DNS requests in transit but does not replace a VPN or hide the device's IP address.
- Emergency Lockdown affects DNS resolution; it is not a full operating-system firewall.
- A valid security-pack signature authenticates the publisher key, not the publisher's trustworthiness.
- Losing both the vault password and recovery code makes encrypted vault backups unrecoverable.

## Verification

- All Go tests pass across the project.
- Go static analysis passes.
- All dashboard JavaScript files pass syntax validation.
- Vault tests cover encryption, tamper detection, password recovery, automatic attempt lockout, and audit-log deletion detection.

See the [feature documentation](../features/README.md) for configuration and usage instructions.
