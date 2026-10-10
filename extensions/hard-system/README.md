# DNS Filter Hard System Extension

The Hard System browser extension connects supported Chromium browsers to the
local DNS Filter service. It provides local webpage risk scoring and browser-side
Focus Capsule enforcement. It does not replace DNS filtering and it does not work
when the DNS Filter service is stopped.

## Supported browsers

- Brave
- Google Chrome
- Other Chromium browsers that support Manifest V3 unpacked extensions

Firefox is not an official target for this extension build.

## Install for development or beta testing

1. Start DNS Filter and confirm that `http://127.0.0.1:8080/healthz` works.
2. Open the browser's extension management page:
   - Brave: `brave://extensions`
   - Chrome: `chrome://extensions`
3. Enable **Developer mode**.
4. Choose **Load unpacked**.
5. Select the repository folder `extensions/hard-system`.
6. Pin **DNS Filter Hard System** to the browser toolbar.

After pulling or installing extension changes, return to the extension page and
press **Reload**. Refresh already-open website tabs afterward.

## Hard System

The extension reads the visible page title, description, and a bounded amount of
visible text, then sends that data only to the local DNS Filter API for scoring.
If the score reaches the configured threshold, the page is replaced with the
local Hard System warning screen.

## Focus Capsules

Create a capsule from **Discipline → Command Center → Focus Engine → Manage**.
A capsule can contain:

- allowed website domains;
- allowed YouTube channel names, handles, IDs, or URLs;
- allowed executable names or full paths;
- strict or non-strict website behavior;
- Motivation, Roast, Meme, or Strict block-page language.

When no YouTube channel is configured, the first video's owner can become the
session's learned channel. Other videos from that channel remain available while
the YouTube homepage, Shorts, unrelated channels, and strict out-of-scope sites
can be blocked.

The **Allow this page once** button applies only to the current tab session. End
the Focus session from Discipline to remove the capsule's temporary controls.

## Privacy and security

- Extension requests go only to `127.0.0.1:8080` or
  `dnsfilter.localhost:8080`.
- DNS Filter accepts extension decisions only from loopback and expected browser
  extension origins.
- No keystrokes, passwords, message contents, or browser history database are
  collected by this extension.
- Website text submitted for Hard System scoring stays on the local service.
- AI Session Coach runs only when the user requests it from saved Focus history.

## Troubleshooting

### The extension says the service is unavailable

Open `http://127.0.0.1:8080/healthz`. If it does not return a healthy response,
start or restart DNS Filter before reloading the extension.

### A Focus Capsule does not block the current tab

1. Confirm that a Focus session is active and a capsule is selected.
2. Reload the extension from the browser extension page.
3. Refresh the website tab.
4. Confirm that the site is not listed in the capsule's allowed domains.

### The wrong YouTube channel was learned

End the current Focus session, open the intended video, and start a new session.
Alternatively, edit the capsule and enter the expected channel handle or channel
URL explicitly.

### Emergency recovery

End the Focus session from Discipline first. If application network rules remain,
use the platform's `reset-app-network` recovery command from the installation
guide.
