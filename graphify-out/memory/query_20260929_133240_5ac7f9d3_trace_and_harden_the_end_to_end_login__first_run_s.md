---
type: "query"
date: "2026-09-29T13:32:40.288821+00:00"
question: "Trace and harden the end-to-end login, first-run setup, Vault unlock, account recovery, session invalidation, and recovery-key UX before release."
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", "Config", "SecurityConfig"]
---

# Q: Trace and harden the end-to-end login, first-run setup, Vault unlock, account recovery, session invalidation, and recovery-key UX before release.

## Answer

Expanded from graph vocabulary: auth, login, password, setup, unlock. Added a full temporary E2E flow covering first-run account/Vault creation, lock, protected-action rejection, wrong/correct unlock, recovery-key rotation, stale-session invalidation, replacement-password login, logout, and rejection of the old recovery key. Account recovery now enforces the full password policy. Vault recovery failures now lock out, nil Vault endpoints return 503, and the WPF desktop displays legacy one-time recovery keys in a mandatory save/copy dialog. Go, WPF, JavaScript, and live health checks passed.

## Outcome

- Signal: useful

## Source Nodes

- Server
- Config
- SecurityConfig