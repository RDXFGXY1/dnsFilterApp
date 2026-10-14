#!/usr/bin/env bash
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "Run with sudo: sudo ./emergency-restore-dns-arch.sh" >&2
  exit 1
fi

echo "DNS Filter - Emergency DNS Recovery"
echo "Stopping DNS Filter and restoring automatic DNS without deleting user data..."

# Stop route repair before changing DNS, then release only DNS Filter-owned
# network tables. Missing services/tables are expected during partial installs.
systemctl disable --now dns-filter.service >/dev/null 2>&1 || true
nft delete table inet dnsfilter_apps >/dev/null 2>&1 || true
nft delete table inet dnsfilter_freeze >/dev/null 2>&1 || true

restored=0

if command -v resolvectl >/dev/null 2>&1 && resolvectl status >/dev/null 2>&1; then
  while IFS= read -r iface; do
    [[ -n "$iface" && "$iface" != "lo" ]] || continue
    resolvectl revert "$iface" >/dev/null 2>&1 || true
  done < <(ip -o route show default | awk '{print $5}' | sort -u)
  systemctl restart systemd-resolved.service >/dev/null 2>&1 || true
  restored=1
fi

if command -v nmcli >/dev/null 2>&1 && [[ "$(nmcli -t -f RUNNING general 2>/dev/null || true)" == "running" ]]; then
  while IFS= read -r connection; do
    [[ -n "$connection" ]] || continue
    nmcli connection modify "$connection" ipv4.dns "" ipv4.ignore-auto-dns no ipv6.dns "" ipv6.ignore-auto-dns no
    nmcli connection up "$connection" >/dev/null
  done < <(nmcli -t -f NAME,DEVICE connection show --active | awk -F: '$2 != "" {print $1}')
  restored=1
fi

if [[ -f /etc/resolv.conf.dnsfilter.bak ]]; then
  cp -f /etc/resolv.conf.dnsfilter.bak /etc/resolv.conf
  rm -f /etc/resolv.conf.dnsfilter.bak
  restored=1
fi

if [[ $restored -ne 1 ]]; then
  echo "Automatic DNS could not be identified. Use your network manager to set IPv4 and IPv6 DNS to Automatic." >&2
  exit 1
fi

if grep -Eq '^[[:space:]]*nameserver[[:space:]]+(127\.0\.0\.1|::1)([[:space:]]|$)' /etc/resolv.conf 2>/dev/null; then
  echo "DNS still points to localhost. Reconnect the network or reboot, then verify /etc/resolv.conf." >&2
  exit 1
fi

echo "Automatic DNS restored. DNS Filter startup was disabled and user data was not deleted."
echo "To re-enable protection later: sudo systemctl enable --now dns-filter.service"
