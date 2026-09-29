# Filtering, Rules, and Content Protection

## Filtering master switch

Enables or disables rule enforcement while keeping the service and dashboard available. Temporary pause provides timed relief without permanently changing policy.

## Encrypted upstream DNS

Supports DNS-over-HTTPS and DNS-over-TLS upstreams. Secure-only mode rejects plaintext upstream configuration. Encryption protects DNS transport to the configured resolver; it does not create a VPN or hide all traffic.

## System DNS routing

Points Windows DNS at the local filter and can restore automatic DHCP DNS. Health checks verify whether the active adapter is using the expected local route.

## Application Blocker

Provides one-tap DNS bundles for services such as TikTok, Instagram, YouTube, Facebook, X, Reddit, Snapchat, Netflix, Twitch, gaming, and gambling.

## App Network Control

Uses Windows Firewall to block outbound network access for a selected executable without disconnecting the rest of the computer. Rules use stable names derived from normalized paths. Critical Windows processes and the DNS Filter service itself are rejected.

## Categories

Enables content policy groups such as adult content, ads, trackers, gambling, gaming, social media, and other maintained categories. Category matches produce an explainable reason.

## Keyword groups

Matches configured domain keywords. The dashboard shows enabled groups and observed matches. Keywords are useful but can be broader and more error-prone than explicit domain rules.

## Custom blocklist

Always-block rules created by the user. Search, compact/expanded views, and removal controls manage the list.

## Whitelist and wildcard exceptions

Always-allow rules take precedence according to the decision model. Wildcards such as `*.spotify.com` cover the root service and nested subdomains. Observed-subdomain discovery shows concrete domains before or after a wildcard is trusted.

## Blocklist sources

Adds remote hosts/filter sources, toggles them independently, refreshes lists, and schedules automatic updates. Source failure should not erase installed protection.

## AI domain analyzer

Uses local, explainable domain-name signals to flag suspicious or adult-looking patterns. It suggests a rule but does not automatically block solely because the score exists.

## Hard System content analyzer

The browser extension sends limited page text/title metadata to the local service for content scoring. It can identify adult or suspicious page patterns that DNS alone cannot see. It remains separate from the DNS decision engine and must disclose its content-access permissions.

## Block-page behavior

Supports NXDOMAIN, redirect, or local block-page actions. The local block page explains the reason and may accept an unblock request. HTTPS interception requires the local certificate authority and has important trust implications.

## Decision order

The effective outcome considers trusted exceptions, temporary unlocks, app and schedule context, categories, keywords, direct blocks, and AI signals. Decision Center exposes the winning layer rather than hiding precedence.

