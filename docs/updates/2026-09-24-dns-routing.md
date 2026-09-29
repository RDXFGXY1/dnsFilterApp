# DNS routing repair — 2026-09-24

Windows could resolve blocked domains through the router's IPv6 DNS even when
IPv4 pointed to the local filter. Setup and reset now cover both address families
on the selected adapter, with health checks before setup and a DNS cache flush.
The server provides an explicit IPv6 loopback listener for IPv4 configurations.
Route status checks all detected resolver addresses instead of searching only
for the presence of 127.0.0.1.

The AI confidence API already returns a percentage. The dashboard no longer
multiplies that number by 100. The model's suggestion is separate from the final
filter decision, which can block a domain based on categories or other rules.

Verified on the affected machine: default Windows DNS, IPv4 loopback, and IPv6
loopback returned the block redirect for the reported domain; example.com still
resolved normally. Browser secure DNS, VPNs, other adapters, and cached browser
connections remain outside this routing check. Restart the browser after setup;
if needed, configure its Secure DNS setting to use the system resolver.

The updated desktop executable is build/DNS-Filter-Go-routing.exe. Existing server
binary was backed up to build/dns-filter-before-routing-fix.exe before replacement.
