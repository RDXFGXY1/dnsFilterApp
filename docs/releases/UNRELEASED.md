# Unreleased changes after v0.1.0

This is the working changelog for the next release. The source version is still
`0.1.0`; these changes have not been published as a new portable package.
See the [v0.2.0 release draft](v0.2.0-draft.md) for grouped release notes and
remaining checks. The previous release is documented in [v0.1.0](v0.1.0.md).

## Added

- Installable Side Apps: `.adnf` packages with a `direct.json` manifest, an
  independent HTML/CSS/JavaScript workspace, permission review, and a Side Apps
  menu in the main dashboard. Focus OS is the example package and `adnf-pack`
  builds packages from source.
- Windows startup choices, including filtering before sign-in through a system
  task, plus an independent DNS recovery check if the local resolver stops.
- Desktop tray icon and Windows notifications for approaching deadlines.
- Discipline Needs Attention view for overdue tasks, today's deadlines,
  reminders, upcoming work, and a time-based rescue plan.
- Dashboard Domain settings for a user-owned local hostname, SMTP recovery
  settings, and LAN access limited to exact allowed private IP addresses.
- Vault unlock limits, one-time SMTP email recovery codes, styled HTML recovery
  emails, and removable-USB recovery codes.
- Persistent Live DNS history limited to the latest 200 entries, top blocked
  domains, and a timeline of DNS/service health events.
- Live history privacy controls for retention, hidden client IPs, and
  blocked-only storage.
- Network Test custom total data budget (5–500 MB), timed tests (5–60 seconds),
  and a saved history of the latest 30 results with comparison.

## Changed

- Network Test displays ping, jitter, loss, download, and upload with clearer
  metrics and progress instead of the earlier mixed-line chart.
- Windows background startup and local DNS binding handle a port 53 conflict
  with another Windows service more reliably.
- DNS routing is enabled only after the local service reports healthy; Windows
  launch and restart paths account for the system task.
- Dashboard navigation includes the Side Apps entry and a dedicated Domains
  entry under System.

## Fixed

- Dashboard session handling for the desktop application's sign-in flow.
- Vault settings visibility and recovery interactions.
- Discipline Needs Attention loading and task/project sidebar layout.

## Security and scope

- Side Apps use restricted permissions and an isolated workspace; publisher
  identity is self-declared, so installing a package still requires trust.
- SMTP recovery generates a fresh short-lived code at reset time. USB recovery
  works with the saved code while offline.
- Dashboard LAN access requires a configured domain and allowlisted private
  device IPs. Access over the public Internet is a placeholder and is not
  available in this release draft.

These notes describe the current working tree. Packaging, fresh-install,
upgrade, real reboot, and SMTP delivery checks remain release gates.
