---
type: "query"
date: "2026-10-06T21:44:27.046922+00:00"
question: "Design a plugin store for DNS Filter where plugins are separate language-independent local API servers, beginning with TinyPrinter"
contributor: "graphify"
outcome: "useful"
source_nodes: ["Server", ".setupRoutes()", "SecurityConfig", "dashboard.js", "api"]
---

# Q: Design a plugin store for DNS Filter where plugins are separate language-independent local API servers, beginning with TinyPrinter

## Answer

Expanded query to api, routes, server, auth, dashboard, module, config, install, enable, disable, security. Recommended an out-of-process local plugin protocol: DNS Filter owns registry, permissions, UI, proxy, audit, and lifecycle; plugins expose versioned loopback APIs. Dashboard must call DNS Filter proxy because its CSP connect-src self and TinyPrinter has no CORS/auth. Use declarative manifest/UI instead of loading third-party JavaScript. MVP supports loading a manifest from disk and attaching an already-running local server; later store installs only signed packages. TinyPrinter maps status, scan/connect, print, previews, and feed/retract through capability-scoped proxy routes.

## Outcome

- Signal: useful

## Source Nodes

- Server
- .setupRoutes()
- SecurityConfig
- dashboard.js
- api