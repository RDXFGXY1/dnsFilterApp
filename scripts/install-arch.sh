#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/opt/dns-filter"
SERVICE_FILE="/etc/systemd/system/dns-filter.service"
DESKTOP_FILE="/usr/share/applications/dns-filter.desktop"
PRESERVE_DIR="/var/lib/dns-filter-preserved"
BACKUP_ROOT="/var/lib/dns-filter-backups"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -x "$SCRIPT_DIR/dns-filter" && -d "$SCRIPT_DIR/web" ]]; then
  SOURCE_DIR="$SCRIPT_DIR"
else
  SOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
fi
CONFIGURE_DNS=1
INSTALL_DEPS=1
HAD_INSTALL=0

for arg in "$@"; do
  case "$arg" in
    --no-dns) CONFIGURE_DNS=0 ;;
    --skip-deps) INSTALL_DEPS=0 ;;
    *) printf 'Unknown option: %s\nUsage: sudo ./install-arch.sh [--no-dns] [--skip-deps]\n' "$arg" >&2; exit 2 ;;
  esac
done

fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
step() { printf '\n==> %s\n' "$*"; }

[[ $EUID -eq 0 ]] || fail "Run this installer with sudo: sudo ./scripts/install-arch.sh"
[[ -r /etc/os-release ]] || fail "Cannot identify this Linux distribution."
. /etc/os-release
case "${ID:-}:${ID_LIKE:-}" in
  arch:*|manjaro:*|endeavouros:*|*:arch*) ;;
  *) fail "This installer supports Arch Linux and Arch-based distributions only (detected: ${PRETTY_NAME:-unknown})." ;;
esac
command -v systemctl >/dev/null || fail "systemd is required."
command -v pacman >/dev/null || fail "pacman is required."
[[ -d "$INSTALL_DIR" ]] && HAD_INSTALL=1

SERVER_SOURCE="$SOURCE_DIR/dns-filter"
DESKTOP_SOURCE="$SOURCE_DIR/desktop/DNSFilter"
if [[ ! -x "$SERVER_SOURCE" ]]; then SERVER_SOURCE="$SOURCE_DIR/build/dns-filter"; fi
if [[ ! -x "$DESKTOP_SOURCE" ]]; then DESKTOP_SOURCE="$SOURCE_DIR/build/desktop/linux-x64/DNSFilter"; fi
[[ -x "$SERVER_SOURCE" ]] || fail "dns-filter binary not found. Build the Arch package first with: ./scripts/build-arch.sh"
[[ -x "$DESKTOP_SOURCE" ]] || fail "Avalonia desktop binary not found. Build the Arch package first with: ./scripts/build-arch.sh"
[[ -d "$SOURCE_DIR/web" ]] || fail "Web dashboard assets are missing."
[[ -f "$SOURCE_DIR/dns-filter.service" ]] || fail "The systemd service file is missing."
[[ -x "$SOURCE_DIR/arch-health-check.sh" ]] || fail "The Arch health-check helper is missing."
[[ -x "$SOURCE_DIR/arch-diagnostics.sh" ]] || fail "The Arch diagnostics helper is missing."

if [[ $INSTALL_DEPS -eq 1 ]]; then
  step "Installing required Arch runtime libraries"
  pacman -S --needed --noconfirm dotnet-runtime-8.0 fontconfig libx11 libice libsm curl nftables >/dev/null
fi

step "Stopping the previous service safely"
systemctl stop dns-filter.service >/dev/null 2>&1 || true

if [[ -d "$INSTALL_DIR" ]]; then
  BACKUP_DIR="$BACKUP_ROOT/$(date -u +%Y%m%dT%H%M%SZ)"
  install -d -m 0700 "$BACKUP_DIR"
  [[ -d "$INSTALL_DIR/configs" ]] && cp -a "$INSTALL_DIR/configs" "$BACKUP_DIR/"
  [[ -d "$INSTALL_DIR/data" ]] && cp -a "$INSTALL_DIR/data" "$BACKUP_DIR/"
  printf 'Existing configuration and data backed up to %s\n' "$BACKUP_DIR"
fi

step "Installing DNS Filter into $INSTALL_DIR"
install -d -m 0755 "$INSTALL_DIR" "$INSTALL_DIR/configs" "$INSTALL_DIR/data/logs" "$INSTALL_DIR/desktop"
install -m 0755 "$SERVER_SOURCE" "$INSTALL_DIR/dns-filter"
rm -rf "$INSTALL_DIR/web" "$INSTALL_DIR/desktop"
cp -a "$SOURCE_DIR/web" "$INSTALL_DIR/"
install -d -m 0755 "$INSTALL_DIR/desktop"
cp -a "$(dirname "$DESKTOP_SOURCE")/." "$INSTALL_DIR/desktop/"
install -m 0644 "$SOURCE_DIR/configs/config.example.yaml" "$INSTALL_DIR/configs/config.example.yaml"
if [[ $HAD_INSTALL -eq 0 && -d "$PRESERVE_DIR/configs" ]]; then
  cp -a "$PRESERVE_DIR/configs/." "$INSTALL_DIR/configs/"
fi
if [[ $HAD_INSTALL -eq 0 && -d "$PRESERVE_DIR/data" ]]; then
  cp -a "$PRESERVE_DIR/data/." "$INSTALL_DIR/data/"
fi
if [[ ! -f "$INSTALL_DIR/configs/config.yaml" ]]; then
  install -m 0600 "$SOURCE_DIR/configs/config.example.yaml" "$INSTALL_DIR/configs/config.yaml"
fi
if [[ ! -f "$INSTALL_DIR/configs/custom-blocklist.yaml" ]]; then
  install -m 0644 "$SOURCE_DIR/configs/custom-blocklist.yaml" "$INSTALL_DIR/configs/custom-blocklist.yaml"
fi
if [[ -f "$SOURCE_DIR/VERSION" ]]; then
  install -m 0644 "$SOURCE_DIR/VERSION" "$INSTALL_DIR/VERSION"
elif [[ -f "$SOURCE_DIR/internal/version/VERSION" ]]; then
  install -m 0644 "$SOURCE_DIR/internal/version/VERSION" "$INSTALL_DIR/VERSION"
fi
[[ -f "$SOURCE_DIR/README.md" ]] && install -m 0644 "$SOURCE_DIR/README.md" "$INSTALL_DIR/README.md"
install -m 0755 "$SOURCE_DIR/arch-health-check.sh" "$INSTALL_DIR/arch-health-check.sh"
install -m 0755 "$SOURCE_DIR/arch-diagnostics.sh" "$INSTALL_DIR/arch-diagnostics.sh"
chmod 0700 "$INSTALL_DIR/configs" "$INSTALL_DIR/data"

step "Installing and enabling the systemd background service"
install -m 0644 "$SOURCE_DIR/dns-filter.service" "$SERVICE_FILE"

cat > "$DESKTOP_FILE" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=DNS Filter
Comment=Local DNS protection and network analytics
Exec=/opt/dns-filter/desktop/DNSFilter
Icon=network-workgroup
Terminal=false
Categories=System;Security;Network;
StartupNotify=true
DESKTOP

systemctl daemon-reload
systemctl enable --now dns-filter.service
if ! "$INSTALL_DIR/arch-health-check.sh"; then
  journalctl -u dns-filter.service -n 30 --no-pager >&2
  fail "The service did not start. The recent service log is shown above."
fi

if [[ $CONFIGURE_DNS -eq 1 ]]; then
  step "Routing system DNS through the local filter"
  if ! "$INSTALL_DIR/dns-filter" setup-dns; then
    printf 'WARNING: Service installation succeeded, but automatic DNS routing failed.\n' >&2
    printf 'Run later: sudo /opt/dns-filter/dns-filter setup-dns\n' >&2
  fi
fi

printf '\nDNS Filter is installed and starts automatically with Arch Linux.\n'
printf 'Desktop: open “DNS Filter” from your application menu\n'
printf 'Dashboard: http://127.0.0.1:8080\n'
printf 'Status: systemctl status dns-filter.service\n'
printf 'Logs: journalctl -u dns-filter.service -f\n'
printf 'Health: sudo /opt/dns-filter/arch-health-check.sh\n'
printf 'Diagnostics: sudo /opt/dns-filter/arch-diagnostics.sh\n'
