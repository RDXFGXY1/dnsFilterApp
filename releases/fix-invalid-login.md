# Fix: Invalid username or password on /login

Date: 2026-02-21

## Summary

Users reported being unable to log in to the web dashboard using the documented default credentials (`admin` / `changeme`). The server returned `⚠ Invalid username or password` (HTTP 401).

## Cause

The shipped example configuration contained an example bcrypt hash that did not match the documented default password. This caused `bcrypt.CompareHashAndPassword` to fail when users supplied `changeme`.

## Fix

- Runtime compatibility added in `internal/config/config.go`: when the known example bcrypt hash is found in the loaded configuration, the server now regenerates a bcrypt hash for the documented default password (`changeme`) and uses that hash for authentication at startup. This makes the default credentials work on first-run without requiring users to edit the config file manually.

Files changed:
- `internal/config/config.go` — added bcrypt import and compatibility block that generates a bcrypt hash for `changeme` if the example hash is present.

## Verification

1. Restart the server or run it locally:

```bash
go run ./cmd/server/main.go
```

2. Open the login page at `http://127.0.0.1:8080/login` and sign in with:

- Username: `admin`
- Password: `changeme`

3. Alternatively, test the API login endpoint:

```bash
curl -i -X POST http://127.0.0.1:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}'
```

Expected result: HTTP 200 OK and session cookie / JSON success response.

## Notes & Follow-up

- This change updates the runtime behavior only; it does not overwrite `configs/config.yaml` on disk. Consider persisting an updated bcrypt hash during installation for a permanent fix (can be added to `scripts/install.sh`).
- For stronger security, users should change the default password immediately after first login.
