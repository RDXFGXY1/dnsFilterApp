# Product Vision, Scope, and Users

## Vision

The product aims to make local protection understandable. Instead of showing only “blocked” or “allowed,” it connects a decision to evidence, policy, application identity, destination, and a safe next action.

## Core principles

- **Local first:** security data remains on the device by default.
- **Explainable:** important decisions have visible reasons.
- **Recoverable:** quarantine and configuration backup avoid irreversible surprises.
- **Layered:** DNS policy, connection observation, Windows firewall, content signals, and file identity complement each other.
- **User controlled:** automation is configurable and visible.
- **Honest:** unknown does not mean malicious, and a low score does not guarantee safety.

## Intended users

### Personal user

Wants safer browsing, distraction control, encrypted DNS, and simple status information.

### Power user

Wants detailed logs, explicit rules, application-level network control, profiles, exports, and automation.

### Parent or guardian

Wants content-category controls, keyword policies, schedules, wellbeing limits, and unblock requests.

### Security learner

Wants to understand connections, signatures, DNS routes, decision precedence, and risk signals.

## Protection boundaries

The product currently covers DNS filtering, selected Windows firewall controls, connection metadata, content scoring through the Hard System extension, and new-download analysis. It does not claim to replace a VPN, endpoint detection suite, or full antivirus engine.

## Main surfaces

- Go local service: persistent engine and API.
- Web dashboard: complete configuration and investigation interface.
- C# WPF desktop app: native Windows control center.
- Browser extension: optional page-content and download-source metadata.
- Block page: local explanation and unblock-request experience.
- Discipline workspace: productivity and execution system.

