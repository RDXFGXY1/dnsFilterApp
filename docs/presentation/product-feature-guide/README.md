# DNS Filter — Complete Product Feature Encyclopedia

This documentation set explains the complete DNS Filter product for presentations, demonstrations, onboarding, planning, and technical review. It covers the web dashboard, Windows desktop application, DNS engine, monitoring, protection, automation, hard-security controls, system administration, productivity tools, and supporting platform architecture.

The guide describes current behavior from the repository as of September 2026. Proposed ideas are labeled **Planned**. A feature is never described as complete merely because it appears in a design document.

## Documentation map

| Folder | Subject |
|---|---|
| [00-product](00-product/vision-scope-and-users.md) | Product vision, users, security model, and terminology |
| [01-dashboard](01-dashboard/dashboard-overview-and-navigation.md) | Main dashboard, navigation, status, analytics, and decision workflow |
| [02-monitoring](02-monitoring/monitoring-and-intelligence.md) | Live DNS, connection intelligence, destinations, privacy, downloads, and route monitoring |
| [03-protection](03-protection/filtering-rules-and-content-protection.md) | Filtering engine, policies, rules, categories, keywords, AI signals, and block pages |
| [04-automation](04-automation/focus-profiles-schedules-and-learning.md) | Focus, wellbeing, schedules, profiles, learning mode, and automatic response |
| [05-hard-security](05-hard-security/vault-lockdown-quarantine-and-integrity.md) | Vault, lockdown, DNS bypass protection, quarantine, signatures, and recovery |
| [06-system](06-system/settings-authentication-and-maintenance.md) | Settings, authentication, recovery, system DNS, startup, cache, updates, and requests |
| [07-productivity](07-productivity/organizer-and-discipline.md) | Todos, notes, habits, planning, goals, calendar, projects, and analytics |
| [08-desktop](08-desktop/windows-desktop-application.md) | Native C# application, live controls, notifications, and local-service integration |
| [09-platform](09-platform/architecture-data-and-api.md) | Architecture, database, local API, privacy, permissions, and lifecycle |
| [10-presentation](10-presentation/slide-plan-and-demo-script.md) | Presentation structure, demo order, and speaking notes |
| [11-status](11-status/implementation-matrix.md) | Current implementation status and important limitations |
| [Glossary](GLOSSARY.md) | Definitions used across the product |

For the deeper engineering design of Download Guardian, use the separate [Download Guardian presentation package](../download-guardian/README.md).

## Product in one paragraph

DNS Filter is a local-first Windows protection and control platform. Its Go service filters DNS traffic, applies policies, observes network behavior, manages Windows firewall actions, and stores local evidence. Its web dashboard provides detailed investigation and configuration, while its C# desktop application provides native status, analytics, notifications, and essential controls. The product emphasizes explainable decisions, reversible actions, and privacy-preserving local operation.

## Main user journey

1. Sign in to the local service.
2. Verify that filtering and the Windows DNS route are active.
3. Observe DNS queries, blocked activity, application connections, and downloads.
4. Investigate why something was allowed, blocked, or marked risky.
5. Create a rule, trust a destination, quarantine a file/application, or repair protection.
6. Use automation for schedules, profiles, focus, and repeated security responses.
7. Review history and analytics to improve future policy.

