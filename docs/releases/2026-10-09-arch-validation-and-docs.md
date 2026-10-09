# Arch validation and documentation structure

Release date: 2026-10-09

## Verified on a real Arch installation

- The DNS service installs and runs on Arch Linux.
- The dashboard and local DNS protection are available after installation.
- `dns-filter.service` starts protection again after a full reboot.

## Changed

- Connection Freeze now ends its Linux output chain with a verified catch-all
  `counter drop` rule after established-flow and essential-service allowances.
- Connection Freeze reports an active state only after the tagged nftables rule
  can be read back from the kernel.
- Arch diagnostics now checks nftables access and reports whether an active
  Connection Freeze rule is complete.
- Documentation is organized into platform installation, feature, release,
  presentation, and license areas.
- Windows and Arch installation guides now have separate platform READMEs.
- Previous changelogs and dated updates now live together in `docs/releases`.

## Validation still required

- Re-test the corrected Connection Freeze behavior on the real Arch machine.
- Test App Network Control with a non-critical graphical application and verify
  Allow/recovery after restarting that application.
- Validate the ARM64 package on real ARM64 Arch hardware.

## Recovery

If a native network-control test fails:

```bash
sudo /opt/dns-filter/dns-filter reset-app-network
sudo nft delete table inet dnsfilter_apps
sudo nft delete table inet dnsfilter_freeze
```

These nftables tables are private to DNS Filter and do not replace user or
distribution firewall tables.
