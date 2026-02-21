# Releases / Changelog

This folder is used to track release notes, bug fixes, and new features for the project.

Guidelines
- Add entries to `UNRELEASED.md` while you work on changes.
- When you're ready to publish a release, copy relevant sections from `UNRELEASED.md` to a new file named `vX.Y.Z.md` (for example `v1.0.0.md`) and update the version header/date.
- Keep each released file focused and small: list Added, Changed, Fixed, and Deprecated items.

Files
- `UNRELEASED.md` - Working area for ongoing changes.
- `v1.0.0.md` - Example initial release notes (existing baseline).

Semantic versioning
- We recommend using semantic versioning: `MAJOR.MINOR.PATCH`.

Example workflow
1. Work on a bug or feature; append notes to `UNRELEASED.md` under the appropriate heading.
2. When ready, cut a release file `vX.Y.Z.md` and move the items from `UNRELEASED.md` into it.
3. Tag the release in git (optional): `git tag -a vX.Y.Z -m "Release vX.Y.Z"` and `git push --tags`.

Thank you for keeping clear release notes — they make it much easier to track changes and communicate updates to users.
