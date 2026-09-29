# Feature Documentation

This directory contains the permanent documentation for features added to DNS Filter.

## Security and privacy features

- [Hard System](HARD-SYSTEM.md) — score visible webpage content locally and cover high-risk pages with a browser extension.
- [App Network Control](APP-NETWORK-CONTROL.md) — block outbound Windows network access for one selected executable.
- [Connection Freeze](connection-freeze.md) — preserve existing flows and temporarily block new outbound connections on Windows, with application logs.
- [Go Desktop Control](desktop-control.md) — start the server and configure local DNS from Windows.

- [Security Center](SECURITY-CENTER.md) — review the readiness and status of every security feature from one dashboard.
- [Dashboard Account Recovery](ACCOUNT-RECOVERY.md) — reset a forgotten password with a one-time offline recovery key.
- [Encrypted DNS](ENCRYPTED-DNS.md) — send upstream DNS requests through DNS-over-HTTPS or DNS-over-TLS.
- [Domain Decision Explanations](DOMAIN-EXPLANATIONS.md) — understand why a domain was allowed or blocked.
- [Emergency Lockdown](EMERGENCY-LOCKDOWN.md) — temporarily block every domain except explicitly trusted exceptions.
- [Signed Security Packs](SECURITY-PACKS.md) — verify and install cryptographically signed domain lists.
- [Secure Configuration Vault](SECURE-CONFIGURATION-VAULT.md) — protect configuration changes and encrypted backups with a second password.
- [Local Web Security](LOCAL-WEB-SECURITY.md) — protect the local dashboard against forged hosts, cross-site actions, framing, and unsafe browser capabilities.
- [Safe Windows Updates](SAFE-WINDOWS-UPDATES.md) — preserve local state and automatically roll back an unhealthy update.
- [Safe Windows Uninstall](SAFE-WINDOWS-UNINSTALL.md) — restore network state and remove owned resources while preserving user data by default.
- [Windows Lifecycle Resilience](WINDOWS-LIFECYCLE-RESILIENCE.md) — retain explicit DNS-routing intent across restart, resume, and adapter changes.

## Adding future feature documentation

Create one Markdown file per feature and add it to this index. A feature document should explain:

1. What the feature does.
2. Where to find it in the application.
3. How to configure and use it.
4. Its security or privacy guarantees.
5. Its limitations and recovery procedure.

Release-specific summaries belong in [`../updates`](../updates/README.md).
