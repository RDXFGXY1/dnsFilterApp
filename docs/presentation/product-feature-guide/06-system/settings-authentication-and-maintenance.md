# Settings, Authentication, and Maintenance

## Local authentication

The service uses an administrator username, bcrypt password hash, HttpOnly session cookie, session expiration, and request authentication for protected APIs.

## Account recovery

The one-time recovery key resets the login password and rotates recovery material. Attempts are rate-limited and can trigger temporary lockout. The key should be stored offline.

## Password change

Authenticated users can change the administrator password. The configuration stores a hash rather than plaintext.

## Start on Login

Registers or removes elevated Windows startup behavior so protection can launch when the user signs in.

## Filtering controls

- Master enable/disable.
- Timed pause.
- Detection-module toggles.
- Block-action selection.
- Test a domain without browsing to it.

## DNS controls

- Show current adapter and DNS route.
- Set the local filter as Windows DNS.
- Restore automatic DNS.
- Recheck route health.

## Maintenance

- Update all blocklists.
- Reload custom lists.
- Clear DNS cache.
- Restart or repair the local service.
- Show version, ports, and system information.

## Block-page settings

Configure title, message, visual options, redirect behavior, and certificate download. HTTPS block-page support requires careful certificate handling.

## Requests queue

Users submit unblock requests from the block experience. Administrators review, annotate, approve and whitelist, or deny each request.

## Local dashboard address

The product supports `dnsfilter.localhost` as a memorable local name while preserving `127.0.0.1:8080` as the direct service address.

## Data retention

Connection history retention is configurable. Download events and other logs should follow separate documented retention policies as the feature matures.

