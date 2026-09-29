# Implementation Matrix

## Current and operational

| Area | Feature | Status |
|---|---|---|
| DNS | Local DNS server and filtering | Implemented |
| DNS | DoH and DoT upstream support | Implemented |
| DNS | Secure-only upstream validation | Implemented |
| DNS | Windows system DNS setup/reset | Implemented |
| Dashboard | Overview, live monitor, activity, analytics | Implemented |
| Decisions | Inspector, simulator, conflicts, repair | Implemented |
| Rules | Categories, keywords, direct blocks, whitelist | Implemented |
| Rules | Wildcard exceptions and observed subdomains | Implemented |
| Applications | Service bundles | Implemented |
| Applications | Per-executable Windows Firewall control | Implemented |
| Applications | Schedules | Implemented |
| Connections | Timeline, risk signals, details | Implemented |
| Connections | Baselines and unseen destination alerts | Implemented |
| Connections | Reports, retention, notifications | Implemented |
| Connections | Learning mode and profiles | Implemented |
| Hard security | Configuration vault and recovery | Implemented |
| Hard security | Encrypted backup and audit | Implemented |
| Hard security | DNS bypass port-53 protection | Implemented |
| Hard security | Application network quarantine | Implemented |
| Hard security | Emergency DNS lockdown | Implemented |
| Hard security | Signed security packs | Implemented |
| Downloads | Folder monitoring and stability checks | Implemented |
| Downloads | SHA-256, type, name, and signature analysis | Implemented |
| Downloads | Explainable risk scoring | Implemented |
| Downloads | Quarantine and restore | Implemented |
| Downloads | Web and desktop interfaces | Implemented |
| Automation | Focus, guard, time bank | Implemented |
| Productivity | Todos, notes, goals, habits, projects | Implemented |
| Productivity | Planning, calendar, timetable, analytics | Implemented |
| Desktop | Native WPF control center | Implemented |

## Implemented with important limits

| Feature | Important limit |
|---|---|
| DNS bypass kill switch | Blocks direct port 53; does not globally block arbitrary DoH on HTTPS port 443 |
| Hard System | Requires the browser extension for page-content evidence |
| Download source attribution | Exact URL is not available from filesystem monitoring alone |
| Risk scoring | Prioritizes review; does not prove malware or safety |
| Application quarantine | Isolates network access; it does not sandbox filesystem/process behavior |
| Connection telemetry | Based on observable Windows connection state and may omit short-lived activity |
| HTTPS block page | Requires local certificate trust and careful deployment |

## Planned or future expansion

| Feature | Status |
|---|---|
| Exact browser download URL bridge | Planned |
| Safe archive-content inventory | Planned |
| Optional external reputation provider | Planned; must be opt-in |
| Publisher trust and expiration policies | Planned |
| Continuous downloaded-file integrity watch | Planned |
| Full WFP-based per-connection permission prompts | Research/planned |
| Cross-platform parity for Windows firewall features | Not currently available |

## Definition of “implemented”

In this guide, implemented means code and a user/API path exist in the repository. It does not mean every environment, edge case, or deployment method has been independently certified.

