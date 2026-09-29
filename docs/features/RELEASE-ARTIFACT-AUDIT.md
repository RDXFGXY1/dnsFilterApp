# Release Artifact Audit

The Windows portable build is rejected automatically when its folder or ZIP contains local runtime state or private material.

## What is blocked

- databases, query logs, Vault data, quarantine data, backups, and installed security packs;
- private keys, certificate bundles that may contain private material, environment files, and configuration backups;
- debug-symbol files and developer-specific absolute paths;
- recognizable GitHub and OpenAI-style credential tokens;
- an unsafe default `configs/config.yaml` with completed setup or populated JWT/recovery fields;
- archive entries that attempt to escape the extraction directory.

The audit reports only the failed rule and file path. It never prints the matching secret value.

## Build integration

`scripts/build-portable.ps1` builds release binaries without local source paths, removes desktop debug symbols, creates the portable ZIP, and audits both the assembled folder and the final archive. Any finding stops the build with a non-zero exit code.

The sanitized first-run configuration is intentionally included as `configs/config.yaml`. It is accepted only while setup is incomplete and its JWT and recovery fields are empty. User credentials, Vault contents, generated certificates, databases, and logs are never release inputs.

## Manual use

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\audit-release.ps1 -Path dist\dns-filter-portable
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\audit-release.ps1 -Path dist\dns-filter-portable-VERSION.zip
```

This gate complements code signing and published SHA-256 checksums; it does not replace them.
