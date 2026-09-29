---
type: "query"
date: "2026-09-29T13:11:01.400962+00:00"
question: "Trace every Windows-owned resource DNS Filter creates: DNS adapter changes, firewall rules, scheduled tasks/startup entries, services, certificates, app network rules, connection freeze rules, DNS bypass rules, and installation files. Design a safe uninstaller that removes only owned resources, restores automatic DNS, and keeps user data unless explicitly requested."
contributor: "graphify"
outcome: "useful"
---

# Q: Trace every Windows-owned resource DNS Filter creates: DNS adapter changes, firewall rules, scheduled tasks/startup entries, services, certificates, app network rules, connection freeze rules, DNS bypass rules, and installation files. Design a safe uninstaller that removes only owned resources, restores automatic DNS, and keeps user data unless explicitly requested.

## Answer

Implemented scripts/uninstall-windows.ps1 with PlanOnly, exact-path process stopping, automatic DNS restoration, DNSFilter task/service cleanup, scoped owned firewall prefixes/exact rules, matching-thumbprint local CA removal, immutable-file cleanup, default configs/data preservation, and guarded explicit data deletion. Integrated uninstaller transactionally into updater and portable package; replaced unsafe legacy Windows installer. Parser, preservation plan, destructive guard, and package build passed.

## Outcome

- Signal: useful