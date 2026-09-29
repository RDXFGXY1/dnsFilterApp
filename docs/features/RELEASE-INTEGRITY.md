# Release Integrity

Every portable Windows build produces three public files:

- the versioned `.zip` package;
- a `.zip.sha256` checksum using the common `64-hex-digits`, two spaces, filename format;
- a `.zip.manifest.json` document containing the product, version, exact filename, byte size, SHA-256 digest, creation time, Authenticode state, and release-audit state.

Metadata is generated only after optional Authenticode signing, ZIP creation, and the release privacy audit. The checksum therefore identifies the exact artifact intended for publication.

## Verification

Keep the ZIP, checksum, and manifest in the same folder, then run:

```powershell
.\scripts\verify-release.ps1 -Artifact .\dist\dns-filter-portable-VERSION.zip
```

Verification fails if:

- the checksum format is malformed;
- the checksum names a different file;
- any byte of the ZIP changed;
- the manifest product, filename, size, hash, version, or audit state does not match.

Publish all three files together. Users should obtain the checksum from the official release page, not from the same untrusted source as a suspicious download.
