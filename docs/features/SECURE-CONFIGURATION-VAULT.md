# Secure Configuration Vault

The Secure Configuration Vault adds a second password that protects configuration changes. When the vault is locked, DNS resolution and filtering continue normally, but the dashboard cannot modify settings or lists until the vault is unlocked.

## Enable the vault

Open **Settings → System & Security → Secure Vault** and choose a password containing at least 12 characters.

The application displays a **recovery code only once**. Store it offline in a safe location, separate from the vault password.

The password and master key are never stored as plaintext. The vault uses Argon2id for password-based key derivation and AES-GCM for authenticated encryption. In Settings, choose an unlock time and a request limit, or lock the Vault manually. It locks when the selected limit is reached and applies a temporary lockout after repeated incorrect passwords.

## Backups and recovery

- **Download encrypted backup** creates an encrypted copy of the application configuration.
- **Restore backup** authenticates and decrypts the file before restoring it. Restart the application afterward to apply every restored setting.
- The one-time offline recovery code lets you replace a forgotten Vault password.
- You can configure a custom SMTP host and recovery mailbox in the dashboard.
  A reset request generates a new code that expires after 10 minutes. Email
  recovery needs working Internet and SMTP delivery; save the offline code too.
- A removable USB drive can hold a separate recovery code. That saved code can
  reset the Vault password without an Internet connection. Keep the drive
  private: someone with the code and dashboard access could reset the password.

Backups are tied to the master key of the Vault that created them. If the Vault
password and every recovery code are lost, and email recovery is unavailable,
the encrypted backup cannot be recovered.

## Tamper-evident audit log

DNS Filter records configuration changes without storing passwords or request bodies. Audit records are joined by an HMAC chain and protected by an authenticated head record. Modifying, removing, truncating, or reordering recorded entries causes integrity verification to fail.

Use **Verify audit log** in the vault panel to verify the chain and display recent events.

## Configuration

```yaml
vault:
  directory: ./data/vault
  unlock_minutes: 15
  max_failures: 5
  lockout_minutes: 5
```

Do not manually edit `data/vault/vault.json`, `audit.jsonl`, or `audit.head`. Use the dashboard's encrypted backup function instead of copying individual vault files as a configuration backup.
