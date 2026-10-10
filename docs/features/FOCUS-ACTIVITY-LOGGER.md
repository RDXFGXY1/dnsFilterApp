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

Focus History now includes an explicit **AI coach** action. It sends the selected structured session record to the enabled provider only when the user requests it and the Secure Vault is unlocked. Without a configured provider, the app creates a deterministic local review. The resulting review is saved to Intelligence Inbox. Model use remains visible and optional.

Focus Capsules extend this logger with browser mission rules, crash recovery, and temporary application network allowlists. See [Focus Capsules](FOCUS-CAPSULES.md).
