# Focus OS — independent side app

This is a bare starter application with its own HTML, CSS, JavaScript, focus timer, notes, and session history. It does not call Discipline APIs or change DNS rules. Build it out here without changing DNS Filter's core.

- `app/direct.json`: app identity, version, entry point, permissions.
- `app/index.html`: the app's interface.
- `app/styles.css`: its own design.
- `app/app.js`: its own behavior.

Build from the repository root:

```
go run ./cmd/adnf-pack -source ./side-apps/focus-os/app -output ./dist/side-apps/focus-os.adnf
```

In the main DNS Filter dashboard, expand **Side Apps → Add app**, choose `focus-os.adnf`, review the storage permission, and install. Click **Focus OS** in the same menu to launch it.

The host injects the `ADNSF` SDK. Wait for `ADNSF.ready`, then use `ADNSF.storage.get()` and `ADNSF.storage.set(jsonValue)`. The 1 MB storage document belongs only to this app. There is no direct host DOM, cookie, filesystem, network, or DNS API access. A side app starts inside the dashboard, not as an elevated native executable.

Closing or switching away from the app stops its timer. Save notes and finish a session before closing. The full study-system proposal is not implemented in this starter.
