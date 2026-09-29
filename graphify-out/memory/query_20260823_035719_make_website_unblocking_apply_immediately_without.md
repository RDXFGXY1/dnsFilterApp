---
type: "query"
date: "2026-08-23T03:57:19.720400+00:00"
question: "Make website unblocking apply immediately without restarting the browser"
contributor: "graphify"
outcome: "useful"
source_nodes: ["DNSCache", ".handleBlockedDomain()", "BlockPageServer", ".ClearCache()", "WhitelistItem"]
---

# Q: Make website unblocking apply immediately without restarting the browser

## Answer

Diagnosed a 300-second blocked DNS TTL, cache invalidation occurring before whitelist mutation, missing cache clears for custom block mutations, and a cacheable 301 block-page redirect. Fixed blocked redirect TTLs to 1 second; changed whitelist mutation to happen before DNS cache invalidation; added cache invalidation to custom block add/remove; made block pages no-store; removed the permanent redirect; and added a temporary reverse-proxy bypass for whitelisted domains so already-cached 127.0.0.1 browser answers reach the real origin until client DNS state expires.

## Outcome

- Signal: useful

## Source Nodes

- DNSCache
- .handleBlockedDomain()
- BlockPageServer
- .ClearCache()
- WhitelistItem