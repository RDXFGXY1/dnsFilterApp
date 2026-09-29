---
type: "query"
date: "2026-09-29T13:01:50.434576+00:00"
question: "Trace HTTP server startup and every API endpoint that accepts uploaded files or large request bodies. Where should timeouts, header limits, and safe per-route body limits be added without breaking Security Packs or imports?"
contributor: "graphify"
outcome: "useful"
---

# Q: Trace HTTP server startup and every API endpoint that accepts uploaded files or large request bodies. Where should timeouts, header limits, and safe per-route body limits be added without breaking Security Packs or imports?

## Answer

Added server ReadHeaderTimeout 5s, ReadTimeout 30s, WriteTimeout and IdleTimeout 60s, MaxHeaderBytes 32 KiB, and a global 6 MiB mutation body cap. Existing smaller Hard System, Security Pack, and Vault upload limits remain layered. Added limit tests; full verification and deployment passed.

## Outcome

- Signal: useful