---
type: "query"
date: "2026-09-29T18:58:17.947823+00:00"
question: "Harden the dashboard Content Security Policy without breaking dashboard, login, setup, or Discipline."
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", "Template", "SecurityConfig"]
---

# Q: Harden the dashboard Content Security Policy without breaking dashboard, login, setup, or Discipline.

## Answer

Expanded from graph vocabulary: content, dashboard, discipline, login, middleware, script, security, setup, template. Replaced broad inline script/style permission with a cryptographically random per-response nonce and a default-deny CSP covering objects, frames, workers, media, manifests, network connections, base URIs, and form targets. Added nonce propagation to dashboard, login, setup, and Discipline templates plus automated nonce uniqueness, policy, and template coverage tests. Legacy event/style attributes remain narrowly isolated under script-src-attr/style-src-attr for compatibility. Full Go/WPF/JS checks passed, live headers matched page nonces, nonces differed per response, and browser recovery-navigation worked with no CSP console errors.

## Outcome

- Signal: useful

## Source Nodes

- Server
- Template
- SecurityConfig