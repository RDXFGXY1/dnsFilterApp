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

The current implementation stores structured local data and is AI-ready. A later model integration should receive an aggregated summary rather than unrestricted raw activity. Model use must remain visible, optional, and separately consented.
