# 5. Quarantine, Recovery, and Security

## Quarantine objective

Quarantine removes a suspicious file from its executable location while preserving enough information for review and recovery. It is not the same as deletion.

## Proposed storage model

Each quarantined item receives a random internal identifier. The stored object should not retain an executable extension.

```text
data/quarantine/
  objects/
    8f/8f70...opaque-object
  manifests/
    8f70...json.enc
```

The manifest records the original path, hash, size, timestamps, reason, score, signer evidence, and quarantine time. Sensitive manifests should be protected using the existing vault encryption capability or an equivalent authenticated-encryption design.

## Safe quarantine transaction

1. Reopen the file and verify that its identity still matches the analyzed hash.
2. Create a database operation with state `preparing`.
3. Move the file to a quarantine object path on the same volume when possible.
4. If a move is impossible, copy to a temporary quarantine path, hash the copy, atomically rename it, then remove the source.
5. Remove inherited execute-friendly naming.
6. Restrict filesystem permissions.
7. Write the encrypted manifest.
8. Commit database state as `quarantined`.
9. Emit an audit event and desktop notification.

If any verification fails, the original file remains and the operation becomes `failed`; the system must not pretend quarantine succeeded.

## Restore transaction

Restore requires:

- an unlocked configuration vault;
- explicit confirmation;
- destination validation;
- a warning when the original directory no longer exists;
- collision handling when another file now occupies the original path;
- hash verification before and after restoration.

The default collision behavior should be “ask,” never silently overwrite.

## Delete transaction

Permanent deletion is separate from quarantine. The user must confirm it. The audit log retains metadata and the final hash, but not the deleted content.

## Preventing quarantine abuse

- Reject paths outside configured monitoring scopes unless manually selected with confirmation.
- Do not follow reparse points blindly.
- Protect against path traversal and alternate data stream confusion.
- Never quarantine the DNS Filter service, desktop client, Windows critical processes, or quarantine storage itself.
- Limit maximum simultaneous operations.
- Require exact database identity for restore and delete.

## Recovery after a crash

On startup, the manager examines operations left in `preparing`, `moving`, or `restoring` states. It compares source, destination, manifest, and hashes before choosing a safe recovery action. Ambiguous states are surfaced to the user rather than guessed.

## Interaction with Microsoft Defender

Download Guardian should complement Defender. It can record Defender availability and scan outcomes when a stable supported integration is used, but it must not disable Defender or claim that a signature check is equivalent to an antivirus scan.

