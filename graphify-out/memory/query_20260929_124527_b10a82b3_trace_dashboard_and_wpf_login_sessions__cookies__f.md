---
type: "query"
date: "2026-09-29T12:45:27.497198+00:00"
question: "Trace dashboard and WPF login sessions, cookies, fetch or HttpClient mutation requests, API middleware ordering, and extension routes so CSRF and Host validation can be added without breaking clients."
contributor: "graphify"
outcome: "useful"
---

# Q: Trace dashboard and WPF login sessions, cookies, fetch or HttpClient mutation requests, API middleware ordering, and extension routes so CSRF and Host validation can be added without breaking clients.

## Answer

Added global Host validation and browser security headers; added per-session CSRF tokens to Go login/setup, automatic same-origin fetch injection in both dashboards, and token propagation in the WPF API client. Extension routes remain outside CSRF. Tests and deployment passed.

## Outcome

- Signal: useful