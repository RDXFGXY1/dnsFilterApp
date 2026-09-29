# Vault, Lockdown, Quarantine, and Integrity

## Secure Configuration Vault

Protects sensitive configuration changes with a second password. DNS protection continues while the vault is locked.

### Vault lifecycle

- Initial setup creates a one-time recovery code.
- Unlock grants temporary authorization.
- Auto-lock reduces unattended administrative access.
- Manual lock ends the session immediately.
- Recovery rotates the password and recovery material.
- Repeated failures create a temporary lockout.

### Quick-unlock popup

When a protected action is attempted, the dashboard requests the vault password and retries the exact pending action after successful unlock.

### Encrypted backup and restore

Exports authenticated encrypted configuration and validates it before restoration. Backup files must be stored securely.

### Tamper-evident audit trail

Records protected configuration changes and verifies audit-chain integrity.

## DNS Bypass Kill Switch

Creates Windows Firewall rules that block direct outbound plaintext DNS on UDP/TCP port 53. The local filter continues using encrypted upstreams. HTTPS-based DoH uses port 443 and cannot be globally blocked by port without breaking ordinary web traffic.

## Application Quarantine

Blocks all outbound network traffic for one executable through a dedicated Windows Firewall rule. Quarantine remains after restart until explicitly released.

## Download quarantine

Verifies the analyzed file hash, moves the file to protected storage, changes database state, and supports restore to the original path. If the original path is occupied, restore stops instead of overwriting silently.

## Signed Security Packs

Accepts protection packs signed by a trusted Ed25519 publisher. Users add trusted public keys through a separately verified channel. Packs can be verified without installation.

## Application identity safety

Network rules are tied to full executable paths and stable hashed rule names. Critical system processes and the DNS Filter executable cannot be blocked by normal controls.

## Emergency recovery

- Stop Connection Freeze even when the vault is locked.
- Restore DNS to automatic routing.
- Release application/file quarantine through authenticated controls.
- Repair filtering and clear stale cache state.
- Recover the vault with the offline recovery code.

