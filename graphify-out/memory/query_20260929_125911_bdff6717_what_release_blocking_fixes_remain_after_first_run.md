---
type: "query"
date: "2026-09-29T12:59:11.621083+00:00"
question: "What release-blocking fixes remain after first-run setup, atomic config persistence, CI builds, version alignment, and web security hardening? Prioritize the next concrete fix with affected files."
contributor: "graphify"
outcome: "useful"
---

# Q: What release-blocking fixes remain after first-run setup, atomic config persistence, CI builds, version alignment, and web security hardening? Prioritize the next concrete fix with affected files.

## Answer

Prioritized and implemented login brute-force protection in internal/api/server.go: per-client and username tracking, five-attempt five-minute lockout, Retry-After response, successful-login reset, recovery reset, and corrected lockout minute rounding. Added tests and docs; full Go, WPF, and JavaScript verification passed.

## Outcome

- Signal: useful