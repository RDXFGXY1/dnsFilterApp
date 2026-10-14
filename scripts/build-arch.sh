#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="${1:-x64}"
"$ROOT/scripts/build-linux.sh" "$ARCH"

DIST="$ROOT/dist/dns-filter-linux-${ARCH}"
install -m 0755 "$ROOT/scripts/install-arch.sh" "$DIST/install-arch.sh"
install -m 0755 "$ROOT/scripts/uninstall-arch.sh" "$DIST/uninstall-arch.sh"
rm -f "$DIST/install.sh"
cp "$ROOT/docs/installation/arch/README.md" "$DIST/README.md"
cp "$ROOT/scripts/dns-filter.service" "$DIST/dns-filter.service"
cp "$ROOT/scripts/arch-health-check.sh" "$DIST/arch-health-check.sh"
cp "$ROOT/scripts/arch-diagnostics.sh" "$DIST/arch-diagnostics.sh"
cp "$ROOT/scripts/emergency-restore-dns-arch.sh" "$DIST/emergency-restore-dns.sh"
chmod +x "$DIST/arch-health-check.sh" "$DIST/arch-diagnostics.sh" "$DIST/emergency-restore-dns.sh"
tar -C "$ROOT/dist" -czf "$ROOT/dist/dns-filter-arch-${ARCH}.tar.gz" "dns-filter-linux-${ARCH}"
echo "Arch Linux package ready: $ROOT/dist/dns-filter-arch-${ARCH}.tar.gz"
