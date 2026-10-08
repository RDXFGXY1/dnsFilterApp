# Local Web Security

The dashboard is local, but a malicious website can still try to contact a service running on `localhost`. This protection layer makes those requests much harder to abuse.

## What it protects

- **Trusted address validation:** requests are accepted only through `localhost`, `dnsfilter.localhost`, a loopback IP, or the exact address configured for the dashboard. This reduces DNS-rebinding and forged `Host` attacks.
- **Per-session action token:** every authenticated API action that changes data must present a secret token tied to the current login session. Reading dashboard data does not require this token.
- **Hardened session cookies:** the login cookie is unavailable to JavaScript, uses `SameSite=Strict`, and becomes HTTPS-only when HTTPS is enabled.
- **Browser security policy:** every response receives a cryptographically random CSP nonce. Inline script elements run only when the server rendered that exact nonce; injected script/style elements, external objects, frames, workers, media, manifests, and untrusted network connections are denied.
- **Private API responses:** API responses are marked `no-store` so browsers do not retain sensitive operational data in their cache.
- **Sign-in throttling:** five failed attempts for the same username and client temporarily lock that combination for five minutes. A successful sign-in clears its failure history.
- **Transport limits:** slow headers time out after five seconds, requests have bounded read/write time, headers are limited to 32 KiB, and request bodies are capped at 6 MiB. Upload handlers apply their own smaller format-specific limits as well.

## How it works in the interfaces

After login, the service creates a random session identifier and a separate random action token. The web dashboard reads the non-secret-to-the-page action cookie and automatically attaches it to same-origin write requests. The native Avalonia application on Windows or Linux receives the same token in the successful login response and attaches it to its API actions.

The server compares the session, cookie token, and request header using a constant-time comparison. Missing or mismatched values return `403 csrf_failed`. An untrusted dashboard address returns `421 untrusted_host`.

## Compatibility

The protection covers authenticated `/api/` write operations. First-run setup, login, recovery, static files, safe read requests, and the local Hard System browser-extension endpoint keep their existing flows. A service restart invalidates all sessions, so the user signs in again and receives fresh tokens.

## Current compatibility boundary

The old dashboard and Discipline templates still use inline event and style attributes. Compatibility is restricted to the CSP Level 3 `script-src-attr` and `style-src-attr` directives; broad inline script and style elements are no longer enabled. New inline script blocks must carry the per-response nonce, and an automated template test rejects any block that does not. A later UI cleanup can replace the remaining attributes with delegated event listeners and CSS classes, then change both attribute directives to `'none'`.
