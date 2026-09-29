# Download Guardian — Presentation Documentation

This folder explains the proposed Download Guardian security subsystem for DNS Filter. It is a design package, not a claim that the feature is already implemented.

## Recommended reading order

1. [Problem, Goals, and Product Story](01-problem-and-goals.md)
2. [End-to-End Operation](02-end-to-end-operation.md)
3. [Technical Architecture](03-technical-architecture.md)
4. [Risk Scoring and Decision Engine](04-risk-scoring.md)
5. [Quarantine, Recovery, and Security](05-quarantine-and-security.md)
6. [Database, API, and Desktop UI](06-data-api-and-ui.md)
7. [Privacy, Limitations, and Failure Cases](07-privacy-limitations-and-failures.md)
8. [Implementation Plan and Presentation Script](08-roadmap-and-presentation.md)

## One-sentence explanation

Download Guardian is a local Windows security layer that observes newly downloaded files, records their origin and identity, calculates an explainable risk score, and can isolate dangerous files before they are opened.

## What makes it different

- It joins download evidence with the application's existing DNS and connection history.
- It explains every risk point instead of displaying an unexplained “AI danger” label.
- It works locally and does not upload private files by default.
- Quarantine is recoverable: a file is isolated, not silently destroyed.
- The C# desktop application becomes the main control surface.

## Proposed subsystem status

| Component | Status |
|---|---|
| Go DNS filtering and local API | Existing |
| SQLite storage | Existing |
| C# WPF desktop client | Existing |
| Network connection intelligence | Existing |
| Application firewall quarantine | Existing for applications |
| Download directory watcher | Proposed |
| File identity and signature analyzer | Proposed |
| Download risk engine | Proposed |
| File quarantine vault | Proposed |
| Browser download metadata bridge | Proposed, later phase |

