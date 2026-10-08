# Connection Freeze

Connection Freeze preserves existing network flows and temporarily refuses new
outbound connections. Open **Connection Freeze**, choose manual mode or a duration,
then press **Freeze new connections**. Start may require an unlocked Vault; stop is
kept available for recovery.

## Shared behavior

- Supports manual stop or 1–120 minute sessions.
- Preserves loopback, DNS Filter operation, and DHCP renewal.
- Does not stop offline applications.
- Records enable/disable events and locally retained connection evidence.
- Returns to normal when stopped, expired, or the service performs cleanup.

Do not activate it over a remote administration connection such as SSH.

## Windows backend

Windows uses a dynamic Windows Filtering Platform session at the outbound IPv4 and
IPv6 authorization layers. Existing reauthorized flows remain eligible to continue;
new non-loopback connections are denied. The event feed can include application,
destination, protocol, and timestamp information from WFP.

Dynamic WFP rules disappear when the session closes. No persistent Defender
Firewall profile is replaced.

## Arch Linux backend

Arch uses the private `inet dnsfilter_freeze` nftables table. Its output chain:

- accepts loopback destinations;
- accepts established and related flows;
- keeps DNS, DHCP, ICMP, and ICMPv6 available;
- counts and drops other new outbound packets.

Linux event entries currently report aggregate blocked-packet deltas rather than a
reliable executable/destination attribution. Stopping the freeze deletes only its
private table. The systemd unit also performs table cleanup after service stop.

## Recovery

Restarting the DNS Filter service releases Connection Freeze. On Arch, an emergency
manual cleanup is:

```bash
sudo nft delete table inet dnsfilter_freeze
```

The command affects only the DNS Filter-owned freeze table.

## Limitations

- Existing VPN/proxy connections can continue carrying traffic.
- This is a focus and emergency-control feature, not a boundary against an
  administrator or a process able to delegate traffic.
- It does not replace the operating system firewall or block new inbound traffic.
