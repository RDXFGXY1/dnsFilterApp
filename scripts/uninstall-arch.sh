#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/opt/dns-filter"
PRESERVE_DIR="/var/lib/dns-filter-preserved"
PURGE=0
case "${1:-}" in
  "") ;;
  --purge) PURGE=1 ;;
  *) echo "Usage: sudo ./uninstall-arch.sh [--purge]" >&2; exit 2 ;;
esac

[[ $EUID -eq 0 ]] || { echo "Run with sudo: sudo ./scripts/uninstall-arch.sh" >&2; exit 1; }

if [[ -x "$INSTALL_DIR/dns-filter" ]]; then
  "$INSTALL_DIR/dns-filter" reset-app-network || echo "Warning: release application network rules manually before reboot." >&2
  "$INSTALL_DIR/dns-filter" reset-dns || echo "Warning: restore automatic DNS manually after uninstall." >&2
fi
systemctl disable --now dns-filter.service >/dev/null 2>&1 || true
nft delete table inet dnsfilter_freeze >/dev/null 2>&1 || true
nft delete table inet dnsfilter_apps >/dev/null 2>&1 || true
rm -f /etc/systemd/system/dns-filter.service /usr/share/applications/dns-filter.desktop
rm -f /usr/local/bin/dnscli
systemctl daemon-reload
systemctl reset-failed dns-filter.service >/dev/null 2>&1 || true

if [[ -d "$INSTALL_DIR" && $PURGE -eq 0 ]]; then
  install -d -m 0700 "$PRESERVE_DIR"
  rm -rf "$PRESERVE_DIR/configs" "$PRESERVE_DIR/data"
  [[ -d "$INSTALL_DIR/configs" ]] && mv "$INSTALL_DIR/configs" "$PRESERVE_DIR/configs"
  [[ -d "$INSTALL_DIR/data" ]] && mv "$INSTALL_DIR/data" "$PRESERVE_DIR/data"
  echo "Configuration and data preserved in $PRESERVE_DIR"
fi
rm -rf "$INSTALL_DIR"
if [[ $PURGE -eq 1 ]]; then
  rm -rf "$PRESERVE_DIR" /var/lib/dns-filter-backups
  echo "Configuration, data, and installer backups were purged."
fi
echo "DNS Filter removed. Automatic DNS has been restored."
