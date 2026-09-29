# 6. Database, API, and Desktop UI

## Proposed database tables

### `download_events`

Stores one normalized observation per file identity.

| Field | Purpose |
|---|---|
| `id` | Internal event identifier |
| `path` | Last observed normalized path |
| `filename` | Display filename |
| `sha256` | Content identity |
| `size_bytes` | File size |
| `detected_type` | Header-derived type |
| `extension` | Declared filename extension |
| `source_url` | Exact URL only when known |
| `source_domain` | Normalized hostname when known |
| `source_confidence` | exact, strong, possible, unknown |
| `browser` | Reporting browser when known |
| `signature_status` | valid, invalid, unsigned, unavailable |
| `publisher` | Authenticode signer display name |
| `risk_score` | 0–100 prioritization score |
| `risk_level` | low, review, high, critical |
| `recommended_action` | Engine recommendation |
| `actual_action` | Action taken |
| `state` | pending, analyzing, complete, quarantined, restored, deleted, failed |
| `created_at` | First detection time |
| `updated_at` | Latest transition time |

### Supporting tables

- `download_risk_reasons`: one row per score contribution.
- `download_quarantine`: storage identity and transaction state.
- `download_rules`: extensions, hashes, publishers, and action policies.
- `download_sources`: browser extension correlation records.
- `download_audit`: security-sensitive user and automatic actions.
- `download_settings`: monitored folders, thresholds, retention, and notification policy.

## Proposed local API

```text
GET    /api/downloads
GET    /api/downloads/:id
GET    /api/downloads/analytics
GET    /api/downloads/live
GET    /api/downloads/settings
POST   /api/downloads/settings
POST   /api/downloads/:id/trust
POST   /api/downloads/:id/quarantine
POST   /api/downloads/:id/restore
DELETE /api/downloads/:id
GET    /api/downloads/quarantine
GET    /api/downloads/export?format=csv|json
```

Mutations pass through normal authentication and vault authorization. The live endpoint can use Server-Sent Events or periodic polling; the desktop client should reconnect with backoff.

## Desktop navigation

Add a fixed `Downloads` destination to the C# WPF sidebar. It opens a native desktop view and never redirects to the web dashboard.

## Desktop page layout

### Summary row

- Downloads today.
- Executable files.
- High-risk files.
- Currently quarantined.
- Total downloaded bytes.

### Live stream

Columns:

- time;
- filename;
- source;
- file type;
- publisher;
- score;
- state.

### Details drawer

Selecting an event opens a native details panel containing:

- full path;
- SHA-256 with copy button;
- source attribution and confidence;
- signature and publisher;
- complete risk calculation;
- timeline of state transitions;
- actions appropriate to the current state.

### Analytics

- activity by hour/day;
- risk-level distribution;
- file types;
- top source domains;
- signed versus unsigned executables;
- quarantine, restore, and deletion counts;
- average analysis time;
- incomplete or failed analysis operations.

## Notifications

A notification should be concise:

```text
Download requires review
invoice.pdf.exe · Risk 82/100
Invalid signature and misleading extension
```

Clicking it opens the exact event inside the desktop app, not a browser page.

