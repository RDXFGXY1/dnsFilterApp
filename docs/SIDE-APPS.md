# Side Apps — ADNF v1

Side Apps is a host for independent community-built HTML applications in the **main DNS Filter dashboard**. Expand **Side Apps** in the navbar, choose **Add app**, review a package, and install it. Installed enabled apps appear inside the expandable menu; clicking one launches it in its own workspace. Closing or navigating away destroys its frame. The application is not part of Discipline or another dashboard section.

## Independent project and package

The starter is `side-apps/focus-os/`. Distributable files live in its `app/` folder. Developers edit their own HTML, CSS, and JavaScript there and package them independently. No DNS Filter core changes are required to install another application.

An `.adnf` file is a ZIP archive with this root layout:

```
direct.json
index.html
styles.css
app.js
```

The user imports the `.adnf` file, not raw JSON. `direct.json` is the internal manifest:

```json
{
  "format": 1,
  "id": "my-study-app",
  "name": "My Study App",
  "version": "1.0.0",
  "author": "Your name",
  "description": "What this application does",
  "entry": "index.html",
  "permissions": ["storage"]
}
```

IDs use lowercase letters, digits, and hyphens, begin with a letter, and are 2–64 characters. Versions use `major.minor.patch` with an optional prerelease suffix. Unknown manifest fields and unsupported permissions are rejected. Publisher names are self-declared; v1 does not verify signatures or establish publisher identity.

Build from the repository root:

```
go run ./cmd/adnf-pack -source ./side-apps/focus-os/app -output ./dist/side-apps/focus-os.adnf
```

The packer is also a standalone distributable developer tool: `go build -o adnf-pack.exe ./cmd/adnf-pack` on Windows or `go build -o adnf-pack ./cmd/adnf-pack` on Linux. Developers may use their own ZIP tooling and rename the resulting archive to `.adnf`, as long as `direct.json` is at the archive root and validation passes.

## Runtime and permissions

Apps run in an iframe with `sandbox="allow-scripts"`, without same-origin, forms, popups, top-navigation, downloads, or native execution permissions. The runner also sends an HTTP CSP sandbox, so opening its URL directly does not give app scripts the host origin. The package's script and stylesheet references are bundled into the isolated response; packaged images become data URLs. Network resource loading, `fetch`, workers, objects, and nested frames are blocked by its content security policy. The browser sandbox is not a CPU or memory quota; malicious infinite loops can still affect browser responsiveness. Install apps from sources you trust.

Scripts/styles must use package-relative paths. The initial supported model is ordinary scripts and CSS, not module imports, service workers, CDNs, or dynamic asset fetching. Use inline/data assets for CSS background images and fonts. CSS and JS belong to the app; its markup is never inserted into the dashboard DOM.

The only v1 host permission is `storage`. It gives access to a JSON document up to 1 MB belonging to that app ID:

```js
await ADNSF.ready;
const state = await ADNSF.storage.get(); // null on first launch
await ADNSF.storage.set({ notes: "My app's notes" });
```

The injected SDK uses a dedicated MessageChannel. The parent checks the active frame's source and opaque origin before establishing the channel, exposes only named storage methods, and never accepts arbitrary API paths. The backend checks the installed app, enabled state, and permission on each storage request. App IDs are chosen by the host, not by guest messages. Storage requests are serialized by the host; apps should await each call. There is no direct DNS-control API, host DOM, session cookie, or filesystem access. No native executables are accepted or launched. Future host API capabilities require explicit, separately designed permissions.

## Installation, updates, storage

- Preview parses the archive, validates its resource references, and returns metadata, permissions, and SHA-256 without running app code.
- Install requires the reviewed digest. A changed upload must be reviewed again.
- Existing app IDs require explicit replacement approval; updates preserve that app's local state and may request permissions shown in the new review. Because IDs/authors are self-declared, replacement can replace another author's app with the same ID: inspect the package's source.
- Disable prevents new launches and backend storage access. The current host tab destroys its frame when disabled there; another open tab rechecks when it becomes visible. Existing JS in another visible tab is not synchronously terminated by the backend, but its storage calls are denied.
- Uninstall removes the package and its local data in a transaction after a user confirmation.
- Packages and storage live in `side_app_packages` and `side_app_storage` in the existing local database. Archives are stored as blobs and never extracted into the service filesystem.
- Normal dashboard authentication/CSRF protections apply. Installing/replacing/enabling/removing packages follows the configuration-vault lock; app storage remains usable while the configuration vault is locked.

Validation limits: 5 MB compressed upload, 16 MB total expanded data, 4 MB per file, 128 archive entries, 16 KB manifest. Traversal, absolute/Windows paths, symlinks, case aliases, duplicate paths, and unsupported file extensions are rejected. Allowed files: HTML, CSS, JS, JSON, TXT, PNG, JPEG, GIF, WebP, ICO. Limits prevent oversized packages; they are not a guarantee that app code is benign.

## API

Authenticated host endpoints:

- `GET /api/side-apps`
- `POST /api/side-apps/preview` — multipart `package` file
- `POST /api/side-apps/install` — multipart `package`, `sha256`, `replace`
- `POST /api/side-apps/:id/enabled` — `{ "enabled": true }`
- `DELETE /api/side-apps/:id`
- `GET /api/side-apps/:id/storage`
- `PUT /api/side-apps/:id/storage` — raw JSON document
- `GET /side-apps/run/:id` — authenticated sandboxed runner

The HTTP endpoints are for the trusted dashboard. Guests use the injected SDK. Test coverage includes archive boundaries, permission denial, isolation between app IDs, replacement/data preservation, disable/uninstall behavior, auth/CSRF, digest review, template integration, resource bundling, and runner security headers.

The sandbox design follows the platform's [iframe isolation guidance](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/iframe) and [HTTP CSP sandbox](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/sandbox).

## Verification for this implementation

- Tests passed for `internal/sideapps`, `cmd/adnf-pack`, `internal/api`, and `internal/discipline`; API tests use actual multipart uploads, authentication/CSRF middleware, package review/digest validation, the runner policy, storage, and disabling.
- Avalonia Release build passed with no warnings or errors.
- Browser verification used a disposable test database and temporary loopback server. The Side Apps dropdown survived navbar regrouping, Focus OS launched inside its frame, and saved notes survived closing/reopening the app. The install dialog opened correctly. Browser automation lost its session during the file-chooser upload, so interactive package selection was not verified end-to-end; multipart installation is covered by the API tests.
- The temporary preview server was stopped. Production DNS/service state was not changed. A running older service needs the updated server build and web assets before it can display Side Apps.
