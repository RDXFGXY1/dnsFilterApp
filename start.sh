#!/bin/bash
# DNS Filter - Smart Startup Script

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
RESOLV_BACKUP="/tmp/resolv.conf.dns-filter-backup"

cleanup() {
    [ -f "$RESOLV_BACKUP" ] && sudo cp "$RESOLV_BACKUP" /etc/resolv.conf && rm -f "$RESOLV_BACKUP"
}
trap cleanup EXIT INT TERM

echo -e "${GREEN}"
cat << 'BANNER'
╔═══════════════════════════════════════════════════╗
║         DNS CONTENT FILTER v2.0                  ║
║     Protect your network from harmful content    ║
╚═══════════════════════════════════════════════════╝
BANNER
echo -e "${NC}"

mkdir -p data/logs

# Use Google DNS temporarily so blocklists can be downloaded
echo -e "${YELLOW}[1/4] Setting bootstrap DNS for blocklist downloads...${NC}"
sudo cp /etc/resolv.conf "$RESOLV_BACKUP" 2>/dev/null || true
printf "nameserver 8.8.8.8\nnameserver 1.1.1.1\n" | sudo tee /etc/resolv.conf > /dev/null

echo -e "${YELLOW}[2/4] Starting DNS Filter...${NC}"
sudo ./build/dns-filter &
DNS_PID=$!

echo -e "${YELLOW}[3/4] Waiting for blocklists to load...${NC}"
for i in $(seq 1 90); do
    if grep -q "Blocklist update complete" data/logs/dns-filter.log 2>/dev/null; then
        DOMAINS=$(grep "Blocklist update complete" data/logs/dns-filter.log | tail -1 | grep -oP '\d+(?= total)')
        echo -e "${GREEN}✓ Loaded ${DOMAINS} domains!${NC}"
        break
    fi
    printf "\r  Waiting... %ds" $i; sleep 1
done
echo ""

echo -e "${YELLOW}[4/4] Switching DNS to 127.0.0.1...${NC}"
printf "nameserver 127.0.0.1\n" | sudo tee /etc/resolv.conf > /dev/null

echo -e "${GREEN}"
echo "╔═══════════════════════════════════════╗"
echo "║      DNS FILTER IS NOW ACTIVE!       ║"
echo "║                                       ║"
echo "║  Dashboard: http://127.0.0.1:8080    ║"
echo "║  Login:     admin / changeme          ║"
echo "║  CLI:       ./dns-cli.sh help         ║"
echo "║                                       ║"
echo "║  Press Ctrl+C to stop                 ║"
echo "╚═══════════════════════════════════════╝"
echo -e "${NC}"

wait $DNS_PID
