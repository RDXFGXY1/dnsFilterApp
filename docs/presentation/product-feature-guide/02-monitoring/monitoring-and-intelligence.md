# Monitoring and Intelligence

## Live DNS Monitor

Displays DNS events as they happen. Each row includes domain, query type, client, decision, and time. Search and decision filters narrow the stream. Selecting a query opens its decision explanation.

## Blocked Activity

Shows recent blocked queries with domain and client information. Time-window and text filters help identify repeated behavior. This view focuses on policy outcomes, while Live Monitor includes both allowed and blocked decisions.

## Connection Timeline

Observes established Windows network connections and connects them to application paths, destination IPs, ports, protocols, timestamps, and local risk reasons.

### Application aggregation

Groups many observations by executable so the user can see destination count, observation count, and connection-risk count.

### Connection details

The details view lists individual destinations and observations for one application instead of forcing the user to interpret a single aggregate score.

## New Application Alerts

Detects applications that have not yet been reviewed in connection intelligence. The user can mark an application reviewed or block its outbound access.

## Privacy Baselines

Builds a trusted set of destinations for an application. After the baseline is accepted, unseen destinations can be treated as new and reviewed separately.

## New Connections

Lists destinations that appear after a trusted application baseline. The user can:

- trust only the destination;
- inspect hostname/IP/port/protocol evidence;
- block or quarantine the application;
- filter by risk and time;
- configure Windows notifications;
- export retained history;
- configure automatic response.

## DNS Route Watch

Records the machine's current DNS route and detects when an adapter, DNS server, or protection route changes. A manual check lets the user validate routing immediately.

## Download Guardian

Monitors new files completed in the Windows Downloads folder. It waits for file stability, calculates SHA-256, detects file type, checks Authenticode status, assigns explainable risk, and supports recoverable quarantine and restore.

The dashboard includes live refresh, search, risk/state filters, signature and publisher information, evidence details, risk distribution, and vault-protected actions.

## Reports and retention

Connection evidence supports CSV/JSON export, configurable retention, and manual history clearing. Clearing observational history does not silently remove trusted baselines.

## Monitoring limitations

- DNS cannot see encrypted page contents.
- Connection metadata does not prove malicious intent.
- Exact download URLs require browser metadata.
- Windows may merge or omit repeated connection events.
- Risk scores prioritize review and do not guarantee safety.

