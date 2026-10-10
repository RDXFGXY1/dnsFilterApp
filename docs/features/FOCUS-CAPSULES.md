# Focus Capsules

Focus Capsules turn a Discipline focus session into a reusable local work environment. A capsule can combine a goal or task with browser boundaries, an application network allowlist, a block-page personality, local activity evidence, recovery, and an optional AI review.

## Browser mission lock

The existing `extensions/hard-system` browser extension now handles both Hard System content scoring and Focus Capsule decisions. No second extension is required.

- A capsule can allow selected domains and YouTube channels.
- If no YouTube channel is configured, the first video opened after the session starts becomes the locked channel automatically.
- Later videos from the same channel are allowed.
- The YouTube homepage, Shorts, unrelated channels, and websites outside a strict capsule are replaced with the local focus block page.
- Block pages can use Motivation, Roast, Meme, or Strict language.
- The extension sends only the current URL, title, and extracted YouTube channel identifiers to the loopback service. The decision remains local.

## Application allowlist

When `Allowed applications` is not empty, App Network Control creates temporary operating-system rules for the focus session.

- The allowed list accepts full executable paths or executable names.
- Other eligible running user applications lose outbound network access.
- DNS Filter, critical Windows processes, Linux system daemons, manual application rules, and quarantine rules are kept separate.
- Windows uses program-scoped Windows Firewall rules.
- Arch Linux uses the existing cgroup v2 and nftables backend and targets interactive-user processes.
- Temporary rules are removed when the session ends, when its planned duration expires, or during startup recovery before a still-active capsule is reapplied.

If application enforcement cannot start, the focus session is cancelled and every partial rule is rolled back.

## Crash and restart recovery

Active sessions are persisted in SQLite. After a local service or computer restart, an unexpired session is restored with its capsule, learned YouTube channel, activity logging consent, and remaining duration. Expired sessions are closed automatically.

Both the Discipline manager and App Network Control have independent expiration watchdogs. This prevents a browser crash from leaving temporary application rules active indefinitely.

## AI Session Coach

Every saved focus session exposes an `AI coach` action in Focus History.

- With an enabled AI provider and an unlocked Vault, the configured provider receives the structured session record and returns a constrained review.
- Without a provider, a deterministic local review is generated instead.
- The review is saved to the Discipline Intelligence Inbox.
- DNS observations are described honestly as DNS-visible domains, not proof that a page was actively viewed.

## Safety limits

- `Allow this page once` is intentionally temporary and scoped to the current tab session.
- A Focus Capsule is productivity enforcement, not malware containment.
- Application allowlists should include the browser and tools needed to end or manage the session.
- AI analysis is user-triggered and uses the provider configured in Discipline.
