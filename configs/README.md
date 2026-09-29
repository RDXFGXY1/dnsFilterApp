# Configuration files

- `config.example.yaml` is the public, release-safe template.
- `config.yaml` is the local runtime configuration. It can contain password and
  recovery hashes, trusted signing keys, and machine-specific settings, so Git
  intentionally ignores it.

For a clean installation, copy `config.example.yaml` to `config.yaml`. Release
packaging does this automatically and must never copy a developer's local
`config.yaml`.
