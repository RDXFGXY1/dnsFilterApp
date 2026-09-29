# Dashboard Account Recovery

The automated release test now exercises the complete local flow: first-run account and Vault creation, Vault lock, rejected protected change, incorrect and correct unlock, account recovery, recovery-key rotation, stale-session invalidation, sign-in with the replacement password, and logout. Vault recovery and lockout behavior are tested independently with temporary files.

Account recovery accepts only passwords that satisfy the same policy as first-run setup. Five failed account-recovery attempts cause a 15-minute client lockout. A successful recovery rotates the account recovery key and invalidates every dashboard and desktop session. The old password and old recovery key stop working immediately.

Older installations that receive their first recovery key during a successful login see it once. The web dashboard requires explicit confirmation that it was saved; the Windows desktop now displays a selectable dialog with a copy button and cannot continue until the user confirms the key was stored safely.

DNS Filter creates an offline recovery key after the first successful dashboard sign-in. The key is displayed once and can be copied or downloaded as a text file.

## Recover a forgotten password

1. Open the login page and select **Forgot your password?**
2. Enter the administrator username and saved recovery key.
3. Choose a new password containing at least 12 characters.
4. Save the replacement recovery key shown after the reset.

The previous recovery key is invalidated after every successful reset. A recovery key cannot directly open the dashboard; it can only replace the password. Five invalid recovery attempts trigger a 15-minute lockout.

## Security behavior

- Only a bcrypt hash of the recovery key is stored in `configs/config.yaml`.
- Successful password recovery invalidates every active dashboard session.
- Dashboard sessions use random 256-bit tokens kept in application memory instead of a fixed cookie value.
- Session cookies are HTTP-only and use strict same-site behavior.

If both the password and recovery key are lost, local access to the configuration file and a manual administrator procedure are required. DNS filtering continues to run even when dashboard access is unavailable.
