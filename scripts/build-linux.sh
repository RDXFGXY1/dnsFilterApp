#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARCH="${1:-x64}"
RID="linux-${ARCH}"
DIST="$ROOT/dist/dns-filter-${RID}"

case "$ARCH" in
  x64) GOARCH=amd64 ;;
  arm64) GOARCH=arm64 ;;
  *) echo "Unsupported architecture: $ARCH (use x64 or arm64)" >&2; exit 2 ;;
esac

echo "Building DNS Filter service for $RID..."
mkdir -p "$ROOT/build" "$DIST/configs" "$DIST/desktop" "$DIST/data/logs"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -trimpath -ldflags "-s -w" -o "$DIST/dns-filter" "$ROOT/cmd/server"

echo "Publishing Avalonia desktop for $RID..."
dotnet publish "$ROOT/application/DnsFilter.Avalonia/DnsFilter.Avalonia.csproj" \
  -c Release -r "$RID" --self-contained false -o "$DIST/desktop" \
  -p:DebugType=None -p:DebugSymbols=false

cp "$ROOT/configs/config.example.yaml" "$DIST/configs/config.yaml"
cp "$ROOT/configs/custom-blocklist.yaml" "$DIST/configs/custom-blocklist.yaml"
cp -R "$ROOT/web" "$DIST/web"
cp "$ROOT/internal/version/VERSION" "$DIST/VERSION"
cp "$ROOT/scripts/dns-filter.service" "$DIST/dns-filter.service"
cp "$ROOT/scripts/arch-health-check.sh" "$DIST/arch-health-check.sh"
cp "$ROOT/scripts/arch-diagnostics.sh" "$DIST/arch-diagnostics.sh"
cp "$ROOT/scripts/emergency-restore-dns-arch.sh" "$DIST/emergency-restore-dns.sh"
cp "$ROOT/docs/installation/arch/README.md" "$DIST/RUNNING.md"
cp "$ROOT/docs/installation/arch/TEST-CHECKLIST.md" "$DIST/ARCH-TEST-CHECKLIST.md"
chmod +x "$DIST/dns-filter" "$DIST/desktop/DNSFilter" "$DIST/arch-health-check.sh" "$DIST/arch-diagnostics.sh" "$DIST/emergency-restore-dns.sh"

cat > "$DIST/install.sh" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
if [[ $EUID -ne 0 ]]; then echo "Run with sudo: sudo ./install.sh"; exit 1; fi
install -d /opt/dns-filter
cp -R ./* /opt/dns-filter/
install -m 0644 /opt/dns-filter/dns-filter.service /etc/systemd/system/dns-filter.service
systemctl daemon-reload
systemctl enable --now dns-filter.service
echo "Installed. Desktop: /opt/dns-filter/desktop/DNSFilter"
INSTALL
chmod +x "$DIST/install.sh"

tar -C "$ROOT/dist" -czf "$ROOT/dist/dns-filter-${RID}.tar.gz" "dns-filter-${RID}"
echo "Linux package ready: $ROOT/dist/dns-filter-${RID}.tar.gz"
