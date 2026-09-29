# 8. Implementation Roadmap and Presentation Script

## Recommended implementation phases

### Phase 1 — Reliable observation

- Download settings and database schema.
- Windows Downloads directory watcher.
- Event debounce and stability detection.
- SHA-256, file size, extension, and header identification.
- Native desktop Downloads page.
- Logs and basic analytics.

Exit condition: completed downloads reliably appear once, survive restart, and do not classify partial downloads.

### Phase 2 — Explainable security

- Authenticode inspection.
- Double-extension and type-mismatch detection.
- Deterministic risk engine.
- Risk reason display.
- Configurable thresholds and notifications.
- Unit tests covering benign and suspicious examples.

Exit condition: every score can be reconstructed from stored evidence.

### Phase 3 — Recoverable quarantine

- Quarantine storage and manifest.
- Transactional quarantine, restore, and delete.
- Vault protection and audit trail.
- Crash recovery and integrity verification.
- Desktop review workflow.

Exit condition: no tested failure path silently loses a file or falsely reports success.

### Phase 4 — Network correlation

- Time-window correlation with existing DNS and connection intelligence.
- Source confidence model.
- Website-to-file investigation timeline.
- Domain trust and block evidence in risk explanations.

Exit condition: correlation is useful while clearly distinguishing known from inferred sources.

### Phase 5 — Browser metadata bridge

- Extension permission review.
- Download metadata endpoint with strict schema.
- Per-install authentication token.
- Exact source correlation.
- Chrome/Brave/Edge packaging documentation.

Exit condition: no browsing content, cookies, credentials, or downloaded bytes are collected.

## Testing strategy

- Unit tests for scoring and filename analysis.
- Temporary-directory integration tests for watcher events.
- Tests for files changing during hashing.
- Signed, unsigned, invalid-signature, and renamed-file fixtures.
- Quarantine crash-recovery tests at each transaction stage.
- Path traversal, symlink/reparse-point, and permission tests.
- API authentication and vault-lock tests.
- WPF tests or view-model tests for state rendering.
- Performance tests with large files and archive extraction bursts.

## Suggested presentation structure

### Slide 1 — The gap

“DNS protection can stop a dangerous destination, but after a file reaches the computer, DNS no longer knows what that file becomes.”

### Slide 2 — The idea

“Download Guardian connects network origin, file identity, risk evidence, and a recoverable action in one local timeline.”

### Slide 3 — How it works

Show the sequence diagram from `02-end-to-end-operation.md`.

### Slide 4 — Explainable risk

Show a sample `82/100` decision and its individual contributions. Emphasize that the score is not a probability or unexplained AI output.

### Slide 5 — Safe quarantine

Explain verified move, encrypted manifest, restore, collision handling, and crash recovery.

### Slide 6 — Privacy

State that files remain local by default and that exact browser metadata is optional and narrowly scoped.

### Slide 7 — User experience

Show the proposed native desktop page: summary cards, live downloads, event details, and actions.

### Slide 8 — What it does not claim

It does not replace antivirus, guarantee safety, or magically know an exact source URL without evidence.

### Slide 9 — Roadmap

Present the five phases and their measurable exit conditions.

## Short speaking script

“Our application already protects DNS traffic and observes network behavior. The next security gap appears after a download reaches the device. A browser can show that a file was downloaded, but the user still needs to know whether the file is executable, whether its name is misleading, whether its signature is valid, where it probably came from, and what changed since the last trusted version.

Download Guardian adds a local Windows monitoring layer. It waits for the download to finish, calculates a cryptographic identity, inspects its type and signature, correlates reliable network evidence, and creates an explainable risk score. High-risk files can be moved into a recoverable quarantine instead of being silently deleted. Every action is stored in an audit trail and controlled from the native desktop application.

The key point is honesty: an unknown file is not automatically malware, a signed file is not automatically safe, and a guessed source is never presented as a confirmed URL. The result is a practical security tool that gives the user evidence and control without uploading private files by default.”

## Final project message

Download Guardian changes DNS Filter from a network-only blocker into a broader local protection console while preserving the project's core values: local control, explainable decisions, reversible actions, and visible evidence.
