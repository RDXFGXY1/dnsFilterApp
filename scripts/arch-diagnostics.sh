#!/usr/bin/env bash
set -u

INSTALL_DIR="/opt/dns-filter"
SERVICE="dns-filter.service"
FAILURES=0

ok() { printf '[OK]   %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*"; }
bad() { printf '[FAIL] %s\n' "$*"; FAILURES=$((FAILURES + 1)); }

printf 'DNS Filter - Arch diagnostics\n\n'

if [[ -r /etc/os-release ]]; then
  . /etc/os-release
  case "${ID:-}:${ID_LIKE:-}" in
    arch:*|manjaro:*|endeavouros:*|*:arch*) ok "Supported system: ${PRETTY_NAME:-Arch-based Linux}" ;;
    *) warn "This installation is officially supported only on Arch and Arch-based distributions (${PRETTY_NAME:-unknown} detected)." ;;
  esac
else
  bad "Cannot read /etc/os-release."
fi

command -v systemctl >/dev/null 2>&1 && ok "systemd is available." || bad "systemd is missing."
command -v nft >/dev/null 2>&1 && ok "nftables is available." || warn "nftables is missing; advanced network controls will be unavailable."
[[ -r /sys/fs/cgroup/cgroup.controllers ]] && ok "Unified cgroup v2 is available." || warn "cgroup v2 is unavailable; per-application network control will be disabled."
command -v curl >/dev/null 2>&1 && ok "curl is available." || bad "curl is missing."
[[ -x "$INSTALL_DIR/dns-filter" ]] && ok "DNS Filter service binary is installed." || bad "Service binary is missing from $INSTALL_DIR."
[[ -x "$INSTALL_DIR/desktop/DNSFilter" ]] && ok "Avalonia desktop is installed." || bad "Desktop binary is missing."

if command -v systemctl >/dev/null 2>&1; then
  systemctl is-enabled --quiet "$SERVICE" && ok "Service starts automatically at boot." || bad "Service is not enabled."
  systemctl is-active --quiet "$SERVICE" && ok "Service is running." || bad "Service is not running."
fi

if command -v curl >/dev/null 2>&1; then
  HEALTH="$(curl --silent --show-error --max-time 3 http://127.0.0.1:8080/healthz 2>/dev/null || true)"
  [[ "$HEALTH" == *'"ready":true'* ]] && ok "Local API is healthy." || bad "Local API health check failed."
fi

if command -v ss >/dev/null 2>&1; then
  ss -lntup 2>/dev/null | grep -qE ':(53|8080)[[:space:]]' && ok "DNS/API listeners were detected." || warn "Expected port 53 or 8080 listeners were not detected."
fi

if command -v resolvectl >/dev/null 2>&1; then
  resolvectl status 2>/dev/null | grep -qE 'DNS Servers:[[:space:]]+(127\.0\.0\.1|::1)' && ok "systemd-resolved routes DNS to the local filter." || warn "systemd-resolved does not appear to use the local filter."
elif [[ -r /etc/resolv.conf ]] && grep -qE '^nameserver[[:space:]]+(127\.0\.0\.1|::1)' /etc/resolv.conf; then
  ok "/etc/resolv.conf routes DNS to the local filter."
else
  warn "Could not confirm that system DNS routes through the local filter."
fi

printf '\n'
if [[ $FAILURES -gt 0 ]]; then
  printf 'Diagnostics finished with %d failure(s).\n' "$FAILURES"
  printf 'Recent logs: journalctl -u %s -n 50 --no-pager\n' "$SERVICE"
  exit 1
fi
printf 'Diagnostics passed. Warnings may indicate optional capabilities.\n'
