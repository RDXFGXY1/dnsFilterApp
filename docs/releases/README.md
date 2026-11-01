# Releases and changelog

This is the single home for release notes, bug fixes, and development updates.
Do not create another changelog directory elsewhere in the repository.

Guidelines
- Add entries to `UNRELEASED.md` while you work on changes.
- When you're ready to publish a release, copy relevant sections from `UNRELEASED.md` to a new file named `vX.Y.Z.md` (for example `v1.0.0.md`) and update the version header/date.
- Keep each released file focused and small: list Added, Changed, Fixed, and Deprecated items.
- Use `YYYY-MM-DD-short-title.md` for a dated development update that is not yet a versioned release.

Files
- `UNRELEASED.md` - Itemized changes after the v0.1.0 public release.
- `v0.1.0.md` - First public Windows release and known limitations.
- `v0.2.0-draft.md` - Proposed next release and comparison with the v0.1.0 source baseline; not yet published.
- `v2.6.0-beta.1.md` - Earlier internal cross-platform development preview.
- `v1.0.0.md` - Example initial release notes (existing baseline).
- `2026-10-09-arch-validation-and-docs.md` - Arch boot validation, Connection Freeze correction, and documentation reorganization.

Historical updates
- `2026-10-08-arch-linux-foundation.md`
- `2026-09-privacy-security-suite.md`
- `2026-09-25-classic-dark.md`
- `2026-09-25-aurora-preview.md`
- `2026-09-24-dns-routing.md`
- `2026-09-24-connection-freeze.md`

Semantic versioning
- We recommend using semantic versioning: `MAJOR.MINOR.PATCH`.

Example workflow
1. Work on a bug or feature; append notes to `UNRELEASED.md` under the appropriate heading.
2. When ready, cut a release file `vX.Y.Z.md` and move the items from `UNRELEASED.md` into it.
3. Tag the release in git (optional): `git tag -a vX.Y.Z -m "Release vX.Y.Z"` and `git push --tags`.

Thank you for keeping clear release notes — they make it much easier to track changes and communicate updates to users.
