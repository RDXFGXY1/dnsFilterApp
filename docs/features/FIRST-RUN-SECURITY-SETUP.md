# First-Run Security Setup

A clean installation opens `/setup` instead of accepting a shared default
password. Existing installations remain recognized when they already contain a
custom password or an account recovery hash.

The first phase creates a 3–32 character administrator username, requires a
12–128 character password with at least three character classes, generates an
offline account recovery key, stores only bcrypt hashes, and starts an HttpOnly,
SameSite=Strict local session. The recovery key is displayed once and must be
saved before the dashboard is opened.

The same wizard also creates the Secure Vault. The user can reuse the
administrator password or choose a separate 12–128 character Vault password,
then select a 5, 15, 30, or 60 minute automatic-lock period. The Vault generates
its own recovery code, encrypts its random master key with Argon2id-derived
wrappers and AES-GCM, and starts unlocked for the selected period. The account
recovery key and Vault recovery code are clearly labeled, displayed once, and
downloaded together only when the user requests it.

Setup requests are accepted only from the loopback interface. After the secure
configuration is saved, the endpoint permanently returns a conflict response;
it cannot be used to replace an existing administrator. Login, account recovery,
protected pages, and protected APIs refuse access until setup is complete.

Before credentials can be saved, the wizard runs a loopback-only system
readiness check. The local protection service and DNS listener are required to
be healthy. Administrator rights, Windows Firewall, and the active adapter's
System DNS route are also reported, but remain warnings because DNS routing may
intentionally be enabled after the account and Vault have been secured. The
check is read-only, can be repeated from the page, and closes after onboarding.

The configuration field `jwt_secret` is intentionally empty: this version does
not claim to use JWT. Sessions are cryptographically random, server-side,
memory-resident tokens.
