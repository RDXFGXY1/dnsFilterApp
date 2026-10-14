# Focus Activity Logger

Focus Activity Logger is an explicit opt-in extension to Discipline focus sessions. It creates a local evidence trail that can later power weekly reviews and privacy-preserving AI analysis.

## Data collected during an opted-in session

- The foreground application executable name, sampled every five seconds.
- DNS-visible domain names observed by the local resolver.
- User-recorded distraction labels.
- Session task, planned duration, actual duration, reflection, and focus score.

It does not record keystrokes, passwords, message contents, screenshots, page HTML, or full browser URLs. DNS can identify a domain such as `youtube.com`, but not the specific page or private URL path.

## End-of-session behavior

The user can add a reflection and focus score. A completed session can create a Daily Journal entry automatically. Raw activity remains local in SQLite and is returned only through authenticated Discipline APIs.

## AI analysis boundary

Focus History includes an explicit **Generate AI review** action. Every completed session first receives a reliable local review, so analytics are never lost when the vault is locked, no model is configured, or a provider is temporarily unavailable. On request, the structured session record is sent to the enabled provider. The linked result is shown inside the session details and saved to Intelligence Inbox. The session remembers whether its latest result is AI-generated, local, still analyzing, or failed, and exposes a retry/regenerate action without accumulating duplicate messages.

Focus Capsules extend this logger with browser mission rules, crash recovery, and temporary application network allowlists. See [Focus Capsules](FOCUS-CAPSULES.md).
