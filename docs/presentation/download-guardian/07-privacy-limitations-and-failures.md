# 7. Privacy, Limitations, and Failure Cases

## Privacy model

Default behavior is local-only. The system stores metadata and hashes in the local SQLite database. It does not upload files, URLs, or hashes to third parties unless the user enables a clearly named external reputation provider.

## Sensitive data considerations

Filenames and paths may reveal names, organizations, projects, or medical and financial subjects. Therefore:

- logs must remain local by default;
- exported reports require an explicit action;
- path redaction should be offered during export;
- retention should be configurable;
- browser metadata must exclude cookies, headers, page contents, and form data;
- notification text should avoid displaying sensitive full paths on the lock screen.

## Technical limitations

### Source URL ambiguity

A filesystem watcher cannot reliably identify the original URL. Without an extension, the feature can only correlate nearby network evidence and must label confidence honestly.

### Encrypted and password-protected archives

The system may know that an archive exists but be unable to inspect its contents. This is a review condition, not proof of maliciousness.

### Signed malware

A valid signature proves a relationship to a certificate; it does not prove safety. Stolen or abused certificates exist.

### Unsigned legitimate software

Many legitimate open-source and personal tools are unsigned. Unsigned status adds risk context but should not automatically mean malware.

### Files created outside watched folders

The initial design monitors configured download locations, not the entire disk. Files written elsewhere are outside coverage unless another source reports them.

### Browser cache and internal updates

Browser caches and application self-updaters generate many files. Watching them by default would create noise and performance problems.

## Failure cases and expected behavior

| Failure | Safe behavior |
|---|---|
| Watched directory disappears | Mark watcher degraded and retry; do not crash service |
| Access denied | Record a visible error without claiming the file was scanned |
| File changes during hashing | Discard result and requeue the new identity |
| File deleted before analysis | Record `removed-before-analysis` |
| Database temporarily busy | Retry with bounded backoff |
| Quarantine move fails | Leave original intact and mark operation failed |
| Restore destination exists | Ask for a new path or explicit replacement decision |
| Desktop application closed | Go service continues monitoring and storing events |
| Service restarts | Reconcile pending transactions and resume watchers |
| Browser extension absent | Continue local analysis with unknown/possible source confidence |

## Honest security language

Use:

- “No suspicious signals were found.”
- “This file is signed by …”
- “The source could not be confirmed.”
- “Quarantine completed and was verified.”

Avoid:

- “This file is completely safe.”
- “AI guarantees this is malware.”
- “Downloaded from X” when correlation is only possible.
- “Quarantined” before the move and hash verification succeed.

