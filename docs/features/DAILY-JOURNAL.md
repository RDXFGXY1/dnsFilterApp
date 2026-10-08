# Daily Journal

Daily Journal is a local-first reflection and activity log inside the Discipline console. It supports general daily notes and reusable custom logs such as Studying, Coding, Reading, or Fitness.

## What it includes

- General entries and color-coded custom log categories.
- Title, reflection, date, duration, mood, focus score, tags, and an optional linked project.
- Pinned entries, editing, deletion, full-text search, category filtering, and day filtering.
- A grouped daily timeline and a clickable 28-day activity calendar.
- Total entries, active days, logged time, current/longest streaks, average mood, and average focus.
- Rotating reflection prompts.

## Privacy and storage

Entries are stored locally in SQLite tables named `discipline_journal_entries` and `discipline_journal_types`. Journal content is not sent to an external AI or analytics service.

Deleting a custom category does not delete its entries. Those entries are preserved and moved back to General.

## API

- `GET/POST /api/discipline/journal/types`
- `DELETE /api/discipline/journal/types/:id`
- `GET/POST /api/discipline/journal/entries`
- `PUT/DELETE /api/discipline/journal/entries/:id`
- `GET /api/discipline/journal/stats`

Journal routes use the same authenticated local API boundary as the rest of Discipline.
