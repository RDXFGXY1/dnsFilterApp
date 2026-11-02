# Feature Documentation

This directory contains the permanent documentation for features added to DNS Filter.

## Security and privacy features

- [Hard System](HARD-SYSTEM.md) — score visible webpage content locally and cover high-risk pages with a browser extension.
- [App Network Control](APP-NETWORK-CONTROL.md) — block one executable through Windows Firewall or Arch cgroup v2 plus nftables.
- [Connection Freeze](connection-freeze.md) — preserve existing flows and temporarily block new outbound connections on Windows or Arch.
- [Network Test](NETWORK-TEST.md) — run Quick, Full, data-limited, or timed tests and compare the latest 30 saved results.
- [Go Desktop Control](desktop-control.md) — start the server and configure local DNS from Windows.

- [Security Center](SECURITY-CENTER.md) — review the readiness and status of every security feature from one dashboard.
- [Dashboard Account Recovery](ACCOUNT-RECOVERY.md) — reset a forgotten password with a one-time offline recovery key.
- [Encrypted DNS](ENCRYPTED-DNS.md) — send upstream DNS requests through DNS-over-HTTPS or DNS-over-TLS.
- [Domain Decision Explanations](DOMAIN-EXPLANATIONS.md) — understand why a domain was allowed or blocked.
- [Emergency Lockdown](EMERGENCY-LOCKDOWN.md) — temporarily block every domain except explicitly trusted exceptions.
- [Signed Security Packs](SECURITY-PACKS.md) — verify and install cryptographically signed domain lists.
- [Secure Configuration Vault](SECURE-CONFIGURATION-VAULT.md) — protect configuration, choose unlock limits, and recover with an offline, email, or USB code.
- [Local Web Security](LOCAL-WEB-SECURITY.md) — protect the local dashboard against forged hosts, cross-site actions, framing, and unsafe browser capabilities.
- [Safe Windows Updates](SAFE-WINDOWS-UPDATES.md) — preserve local state and automatically roll back an unhealthy update.
- [Safe Windows Uninstall](SAFE-WINDOWS-UNINSTALL.md) — restore network state and remove owned resources while preserving user data by default.
- [Windows Lifecycle Resilience](WINDOWS-LIFECYCLE-RESILIENCE.md) — retain explicit DNS-routing intent across restart, resume, and adapter changes.
- [Windows Startup](../WINDOWS-STARTUP.md) — choose before-sign-in, after-sign-in, or manual startup and configure DNS fallback.
- [Windows Notifications](../WINDOWS-NOTIFICATIONS.md) — show deadline alerts with the desktop tray companion.
- [Dashboard Domains and LAN](DASHBOARD-DOMAINS.md) — choose a local dashboard hostname and allow specific LAN devices.
- [Live History and DNS Health](LIVE-HISTORY.md) — review recent DNS decisions, blocked domains, and resolver events with privacy controls.

## Discipline features

- [Discipline Task Hub](DISCIPLINE-TASK-HUB.md) — plan goals, projects, tasks, nested steps, time blocks, templates, and recovery work.
- [Daily Journal](DAILY-JOURNAL.md) — keep local general or custom daily logs with calendar and streak analytics.
- [Focus Activity Logger](FOCUS-ACTIVITY-LOGGER.md) — optionally record local app/domain activity during focus sessions for later review.
- [Focus Capsules](FOCUS-CAPSULES.md) — lock a focus session to selected apps, websites, and YouTube channels with crash recovery and AI coaching.
- [Discipline Intelligence Inbox](DISCIPLINE-INTELLIGENCE-INBOX.md) — generate private local briefings or explicit AI-assisted analysis through a Vault-protected provider.
- [Needs Attention](DISCIPLINE-NEEDS-ATTENTION.md) — collect overdue work, reminders, and upcoming deadlines into a rescue view.

## Extension platform

- [Plugin SDK](../PLUGIN-SDK.md) — build API-only plugins with bounded permissions, managed runtimes, native controls, or sandboxed local web interfaces.
- [Side Apps](../SIDE-APPS.md) — package independent HTML applications as `.adnf` files and install them from the dashboard.
- [Focus OS example](../FOCUS-OS.md) — build and install the example Side App.

## Adding future feature documentation

Create one Markdown file per feature and add it to this index. A feature document should explain:

1. What the feature does.
2. Where to find it in the application.
3. How to configure and use it.
4. Its security or privacy guarantees.
5. Its limitations and recovery procedure.

Release-specific summaries belong in [`../releases`](../releases/README.md).
