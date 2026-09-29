# Dashboard Overview and Navigation

## Overview page

The Overview page is the command summary for protection. It combines filtering status, DNS route, blocklist coverage, recent blocks, block rate, protection posture, health warnings, top blocked domains, and progress information.

### Summary cards

- **Queries (24h):** observed DNS volume.
- **Blocked (24h):** DNS decisions that denied access.
- **Block rate:** blocked queries divided by relevant observed queries.
- **Domains on blocklist:** current rule coverage.
- **Protection posture:** combined health indicator, not an antivirus guarantee.

### Timeline chart

Shows blocked activity by time bucket. Peaks help identify a noisy application, repeated retry behavior, or a policy that is too broad.

### Top blocked domains

Ranks repeated domains and exposes patterns such as tracking endpoints, game telemetry, adult content, or broken application dependencies.

## Navigation architecture

The fixed sidebar groups pages by user intention:

- **Dashboard:** Overview and Security Center.
- **Monitor:** Live DNS, Connection Timeline, Alerts, New Connections, Download Guardian, DNS Route Watch, and Activity.
- **Protection:** application policy, categories, keywords, blocklist, whitelist, and AI decisions.
- **Automation:** network profiles, app schedules, Focus, and Wellbeing.
- **Advanced:** Learning Mode, Decision Center, and Connection Freeze.
- **System:** Hard Security, Settings, and Requests.
- **Discipline:** separate productivity workspace.

## Security Center

Security Center converts advanced security features into a readiness view. It checks encrypted DNS, decision explanations, emergency lockdown, signed security packs, and the configuration vault. Its score represents configured protections, not global device safety.

## Decision Center

Decision Center answers “Why is this website allowed or blocked?” It displays the winning policy layer, exact reason, matching direct rules, exceptions, AI/category evidence, conflicts, and recent history.

### Rule simulator

Previews a proposed rule against sample domains before it is saved. It helps detect overbroad patterns.

### Conflict and precedence view

Shows the order in which trusted exceptions, temporary access, application rules, schedules, categories, keywords, direct lists, and AI signals influence a decision.

### Protection repair

Re-enables filtering, clears stale cache state, and attempts to restore local DNS routing when permissions allow.

## Theme and layout

The main product theme is a high-contrast dark interface. Layouts use cards, tables, status badges, search controls, and detailed investigation panels. Responsive rules reduce columns on smaller screens.

