# DNS Filter Plugin SDK (Schema v1)

DNS Filter plugins are separate local applications connected through HTTP. A
plugin may be written in Python, Go, C#, Rust, or any language that can expose
a JSON API. Plugin code is never loaded into the DNS Filter process.

## Current release

Schema v1 supports:

- importing a `plugin.json` manifest from disk;
- persistent install and enable/disable state;
- loopback HTTP servers only (`localhost`, `127.0.0.1`, or `::1`);
- `GET` and `POST` JSON actions;
- declarative panels and fields;
- permission review;
- health checks, timeouts, response limits, and activity logs.
- optional managed runtimes with Start, Stop, Restart, on-demand startup, and
  automatic startup alongside DNS Filter.
- bounded file uploads, image previews, and downloadable plugin responses.
- native manifest-generated controls, embedded local web applications, or both.

Plugins are installed disabled. The configuration vault protects installation,
enable/disable, and removal. Normal plugin actions still require an authenticated
dashboard session and its CSRF token.

Remote catalog downloads and signed packages are intentionally deferred to
later schema versions.

## Minimal manifest

```json
{
  "schema_version": 1,
  "id": "org.example.hello",
  "name": "Hello Plugin",
  "version": "1.0.0",
  "description": "A small local API plugin.",
  "publisher": "Example Publisher",
  "ui": {"mode": "native"},
  "server": {
    "base_url": "http://127.0.0.1:9000",
    "health_endpoint": "/api/status",
    "start_mode": "external"
  },
  "permissions": ["hello.run"],
  "panels": [
    {"id": "main", "title": "Hello", "actions": ["say-hello"]}
  ],
  "actions": [
    {
      "id": "say-hello",
      "label": "Say hello",
      "method": "POST",
      "path": "/api/hello",
      "permission": "hello.run",
      "fields": [
        {"id": "name", "label": "Name", "type": "text", "required": true}
      ]
    }
  ]
}
```

## Manifest rules

- IDs use lowercase letters, numbers, dots, underscores, and hyphens.
- The base URL must use plain HTTP and a loopback host.
- Endpoint paths must begin with `/` and cannot contain traversal, queries, or fragments.
- Every action must reference a declared permission.
- Supported action methods are `GET` and `POST`.
- `timeout_seconds` may be set per action from 1–120 seconds. Actions without
  it use a bounded 45-second compatibility timeout; health checks use 5 seconds.
- Supported fields are `text`, `textarea`, `number`, `select`, and `checkbox`.
- Advanced fields also include `file` and `url`; `cards_lines` transforms
  `question | answer` lines into structured flashcard objects.
- Dotted field IDs produce nested JSON. For example,
  `settings.blackening` becomes `{"settings":{"blackening":4}}`.
- Plugin responses must be JSON and cannot exceed 2 MB.

## Plugin interfaces

`ui.mode` chooses how the Plugin Center presents the plugin:

- `native`: render trusted controls from the manifest and call declared API actions.
- `web`: embed the plugin's existing local web application.
- `both`: offer **Web App** and **Native Controls** tabs.

Web and dual plugins declare a path on their existing loopback server. They
cannot declare a second origin or a remote page:

```json
"ui": {
  "mode": "both",
  "default_view": "web",
  "web_path": "/"
}
```

The dashboard loads that page only after the health endpoint reports ready. It
runs in a sandboxed frame that permits scripts, forms, downloads, and same-origin
storage inside the plugin, but not top-level navigation, popups, or access to the
DNS Filter page. Closing the workspace or stopping the managed service unloads
the frame. Native controls remain the safer fallback for sensitive operations.

### Files and generated content

An action with `body_type: "binary"` sends the chosen file as the raw request
body. Its validated `binary` declaration controls the filename header, optional
metadata header/root, and a maximum size no larger than 50 MB. The dashboard
never gives a plugin an arbitrary local path; it streams only the file selected
by the user.

Actions may declare `response_type: "image"` for an in-workspace preview or
`response_type: "download"` for an explicit local download. Generated content
is capped at 20 MB and continues to pass through the authenticated DNS Filter
backend rather than exposing the plugin service directly to browser code.

## Runtime modes

`server.start_mode` supports four values:

- `external`: DNS Filter never starts or stops the plugin application.
- `manual`: the Plugin Center displays Start, Stop, and Restart controls.
- `on_demand`: opening or invoking the plugin starts its service when needed.
- `automatic`: an enabled plugin starts with DNS Filter.

Managed modes require a platform-specific runtime command. Commands are launched
directly without a shell, and executable and working-directory paths must be
absolute. This keeps quoting predictable and prevents manifest command strings
from being interpreted by PowerShell, CMD, or a Unix shell.

```json
"server": {
  "base_url": "http://127.0.0.1:9000",
  "health_endpoint": "/api/status",
  "start_mode": "on_demand",
  "runtime": {
    "windows": {
      "executable": "C:\\Path\\To\\python.exe",
      "args": ["plugin_server.py", "--no-browser"],
      "working_directory": "C:\\Path\\To\\Plugin"
    },
    "linux": {
      "executable": "/usr/bin/python3",
      "args": ["plugin_server.py", "--no-browser"],
      "working_directory": "/opt/example-plugin"
    },
    "startup_timeout_seconds": 30,
    "shutdown_timeout_seconds": 10
  }
}
```

Process output is saved under `data/logs/plugins/<plugin-id>.log`. DNS Filter
only stops processes it launched itself; externally started services are never
terminated. The install confirmation shows the runtime command before the user
trusts and enables it.

## Language and Go support

The protocol is intentionally language-independent. A Python, Go, C#, Rust, or
Java plugin is the same kind of loopback JSON service. Native Go modules are not
loaded into the DNS Filter process: one plugin crash must not crash DNS filtering,
and a plugin must not receive unrestricted in-process access to private state.

A future Go helper package can reduce boilerplate by providing a loopback server,
health endpoint, typed action registration, permission declarations, graceful
shutdown, and a manifest builder. It should still speak this HTTP protocol. Any
future DNS Filter control API should be capability-based, authenticated per
plugin, disabled by default, and limited to the exact permissions approved by
the user.

## TinyPrinter

The first integration manifest is located at:

```text
integrations/tinyprinter/plugin.json
```

TinyPrinter is configured for on-demand startup in its test manifest. Opening it
from Plugin Center starts the Python web service and waits for `/api/status`.
You can also start it manually without opening its browser:

```powershell
cd C:\Users\ayoub\Documents\MiniPrinter\TiMini-Print
python timiniprint_web.py --no-browser
```

Then open **Plugin Center**, load the manifest, unlock the vault if requested,
and enable TinyPrinter. DNS Filter will connect to `http://127.0.0.1:8765`.
The included manifest contains paths for this development computer; packaged
plugins should generate platform-appropriate absolute paths during installation.

TinyPrinter currently has no API authentication. Keep it bound to loopback. A
future plugin protocol version should provide per-plugin bearer tokens.

## Security notes

DNS Filter does not install plugin JavaScript or HTML and never embeds remote
server addresses. A plugin may opt into showing its existing loopback web page
inside the restricted plugin frame. Native actions continue through the Go
backend, are limited to declared endpoints and fields, and store only action
metadata in the audit log rather than printed text or other content.
