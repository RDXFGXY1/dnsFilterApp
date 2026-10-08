# Discipline Intelligence Inbox

## Purpose

The Intelligence Inbox turns Discipline data into short, actionable briefings. It
looks like an email inbox, but it is local to DNS Filter and does not send or
receive real email.

## Where to find it

Open **Discipline**, then choose **Intelligence Inbox** in the navigation. The
inbox can show all active messages, unread messages, starred messages, and the
archive. Messages can be searched, marked read, starred, archived, or deleted.

## Creating a briefing

Choose **Generate briefing**. DNS Filter builds a limited snapshot containing:

- unfinished tasks and their status, priority, and due date;
- active goals and projects;
- habit names, frequencies, and today's completion state; and
- the ten most recent journal entries, truncated before analysis.

If an enabled AI provider is configured, the default provider analyses that
snapshot and creates one inbox message with a suggested action. If no provider
is configured, DNS Filter creates a deterministic local briefing instead. The
local fallback requires no internet connection and sends no data outside the
computer.

Briefings are generated only after the user requests one. The current version
does not continuously upload Discipline data or silently email it.

## AI Providers Center

Open **AI Providers** from the Intelligence Inbox to add one or more providers:

- OpenAI;
- xAI / Grok;
- Google Gemini; or
- a trusted OpenAI-compatible endpoint.

Each connection has a display name, model, enabled state, and optional default
state. A compatible endpoint must use HTTPS unless it is a loopback address.
Use **Test** to verify the model and credentials before generating a briefing.

Provider API billing is separate from consumer subscriptions such as ChatGPT.
The user must obtain a supported API key from the chosen provider.

## Credential protection

API keys are encrypted through Secure Vault before being stored in the local
database. The Vault must be unlocked to save a new key, test a provider, or use
that provider for a briefing. Editing a provider without entering another key
keeps its existing encrypted key.

The interface never returns the saved plaintext key. Removing a provider also
removes its encrypted credential from DNS Filter.

## Output safety

The provider is instructed to return one concise JSON briefing based only on the
supplied snapshot. DNS Filter validates the response, limits its subject and body
length, normalizes severity, and accepts navigation actions only for known
Discipline views. Invalid provider output is rejected instead of being rendered
as trusted HTML.

## Privacy and limitations

- An AI briefing sends the described snapshot to the selected provider.
- Journal content can contain sensitive information; review the provider's data
  policy before enabling it.
- The result is advice, not an authoritative decision. Verify important dates,
  priorities, and conclusions yourself.
- Provider availability, rate limits, account permissions, model names, and API
  charges are controlled by the provider.
- Deleting an inbox message does not delete data already sent to an external
  provider.

## Recovery

If AI generation fails, unlock Secure Vault, test the selected provider, confirm
the model and API balance, and try again. You can disable or remove the provider
and continue using the private local briefing fallback.
