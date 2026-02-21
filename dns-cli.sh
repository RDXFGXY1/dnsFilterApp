#!/bin/bash
# dns-cli - Interactive DNS Filter Management Shell v2.1
# Auto-detects installation path (/opt/dns-filter or local)

VERSION="2.1"

# ── Auto-detect installation path ────────────────────────────────
if [ -f "/opt/dns-filter/configs/config.yaml" ]; then
    # System-wide installation
    INSTALL_PATH="/opt/dns-filter"
    API="http://127.0.0.1:8080/api"
    CUSTOM_BLOCKLIST="$INSTALL_PATH/configs/custom-blocklist.yaml"
    CONFIG_FILE="$INSTALL_PATH/configs/config.yaml"
elif [ -f "./configs/config.yaml" ]; then
    # Local installation
    INSTALL_PATH="$(pwd)"
    API="http://127.0.0.1:8080/api"
    CUSTOM_BLOCKLIST="./configs/custom-blocklist.yaml"
    CONFIG_FILE="./configs/config.yaml"
else
    echo "Error: Cannot find DNS Filter installation"
    echo "Please run from /opt/dns-filter or project directory"
    exit 1
fi

SESSION_COOKIE="session=authenticated"

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly CYAN='\033[0;36m'
readonly BLUE='\033[0;34m'
readonly MAGENTA='\033[0;35m'
readonly NC='\033[0m'

# History
HISTFILE="$HOME/.dns-cli_history"
touch "$HISTFILE" 2>/dev/null

# Helpers
ok()    { echo -e "${GREEN}✓${NC} $1"; }
err()   { echo -e "${RED}✗${NC} $1"; }
info()  { echo -e "${YELLOW}→${NC} $1"; }
warn()  { echo -e "${YELLOW}!${NC} $1"; }
title() { echo -e "${CYAN}${1}${NC}"; }

is_running() {
    curl -s --connect-timeout 2 "$API/stats" --cookie "$SESSION_COOKIE" >/dev/null 2>&1
}

api_post() {
    curl -s -X POST "$API$1" -H "Content-Type: application/json" ${2:+-d "$2"} --cookie "$SESSION_COOKIE" 2>/dev/null
}

api_get() {
    curl -s -X GET "$API$1" --cookie "$SESSION_COOKIE" 2>/dev/null
}

api_del() {
    curl -s -X DELETE "$API$1" --cookie "$SESSION_COOKIE" 2>/dev/null
}

validate_domain() {
    [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]+)+$ ]]
}

# ── Commands ──────────────────────────────────────────────────────

cmd_block() {
    local domain="$1"
    
    [ -z "$domain" ] && { err "Usage: block <domain>"; return 1; }
    validate_domain "$domain" || { err "Invalid domain: $domain"; return 1; }
    is_running || { err "DNS Filter not running"; return 1; }
    
    # Add or enable in YAML
    if grep -q "domain: \"$domain\"" "$CUSTOM_BLOCKLIST" 2>/dev/null; then
        # Already exists, enable it
        python3 << PYEOF
import re
with open('$CUSTOM_BLOCKLIST', 'r') as f:
    content = f.read()
# Find domain block and set enabled: true
lines = content.split('\n')
result = []
in_domain = False
for line in lines:
    if 'domain: "$domain"' in line:
        in_domain = True
    if in_domain and 'enabled:' in line:
        line = re.sub(r'enabled:\s*(true|false)', 'enabled: true', line)
        in_domain = False
    result.append(line)
with open('$CUSTOM_BLOCKLIST', 'w') as f:
    f.write('\n'.join(result))
PYEOF
        info "Domain exists, set enabled: true"
    else
        # Add new
        cat >> "$CUSTOM_BLOCKLIST" << EOF

  - domain: "$domain"
    category: "custom"
    note: "Blocked via CLI $(date +%Y-%m-%d)"
    enabled: true
EOF
        ok "Added to $CUSTOM_BLOCKLIST"
    fi
    
    # Reload
    local resp=$(api_post "/blocklist/reload-custom")
    if echo "$resp" | grep -q "success"; then
        ok "Blocked: $domain (cache cleared)"
    else
        warn "Added to file. Restart: systemctl restart dns-filter"
    fi
}

cmd_unblock() {
    local domain="$1"
    
    [ -z "$domain" ] && { err "Usage: unblock <domain>"; return 1; }
    is_running || { err "DNS Filter not running"; return 1; }
    
    # Set enabled: false in YAML
    if grep -q "domain: \"$domain\"" "$CUSTOM_BLOCKLIST" 2>/dev/null; then
        python3 << PYEOF
import re
with open('$CUSTOM_BLOCKLIST', 'r') as f:
    content = f.read()
lines = content.split('\n')
result = []
in_domain = False
for line in lines:
    if 'domain: "$domain"' in line:
        in_domain = True
    if in_domain and 'enabled:' in line:
        line = re.sub(r'enabled:\s*(true|false)', 'enabled: false', line)
        in_domain = False
    result.append(line)
with open('$CUSTOM_BLOCKLIST', 'w') as f:
    f.write('\n'.join(result))
PYEOF
        ok "Set enabled: false"
    else
        warn "Domain not in blocklist"
    fi
    
    # Remove from memory + clear cache
    api_del "/custom-blocklist/$domain" >/dev/null 2>&1
    api_post "/system/clear-cache" >/dev/null 2>&1
    
    ok "Unblocked: $domain (cache cleared)"
}

cmd_enable() {
    local domain="$1"
    [ -z "$domain" ] && { err "Usage: enable <domain>"; return 1; }
    cmd_block "$domain"
}

cmd_disable() {
    local domain="$1"
    [ -z "$domain" ] && { err "Usage: disable <domain>"; return 1; }
    cmd_unblock "$domain"
}

cmd_remove() {
    local domain="$1"
    
    [ -z "$domain" ] && { err "Usage: remove <domain>"; return 1; }
    
    if grep -q "domain: \"$domain\"" "$CUSTOM_BLOCKLIST" 2>/dev/null; then
        python3 << PYEOF
with open('$CUSTOM_BLOCKLIST', 'r') as f:
    lines = f.readlines()
result = []
skip = 0
for line in lines:
    if skip > 0:
        skip -= 1
        continue
    if 'domain: "$domain"' in line:
        skip = 3
        continue
    result.append(line)
with open('$CUSTOM_BLOCKLIST', 'w') as f:
    f.writelines(result)
PYEOF
        ok "Removed: $domain"
        is_running && {
            api_del "/custom-blocklist/$domain" >/dev/null
            api_post "/system/clear-cache" >/dev/null
        }
    else
        err "Domain not found"
    fi
}

cmd_whitelist() {
    local domain="$1"
    
    [ -z "$domain" ] && { err "Usage: whitelist <domain>"; return 1; }
    validate_domain "$domain" || { err "Invalid domain"; return 1; }
    is_running || { err "DNS Filter not running"; return 1; }
    
    local resp=$(api_post "/whitelist" "{\"domain\":\"$domain\"}")
    echo "$resp" | grep -q "success" && ok "Whitelisted: $domain" || err "Failed"
}

cmd_test() {
    local domain="$1"
    
    [ -z "$domain" ] && { err "Usage: test <domain>"; return 1; }
    
    title "Testing: $domain"
    
    local result=$(dig @127.0.0.1 "$domain" +short +time=2 2>/dev/null | head -1)
    
    if [ -z "$result" ]; then
        err "BLOCKED ✓ - Returns NXDOMAIN"
    else
        ok "ALLOWED - IP: $result"
    fi
}

cmd_list() {
    [ ! -f "$CUSTOM_BLOCKLIST" ] && { err "Not found: $CUSTOM_BLOCKLIST"; return 1; }
    
    title "Custom Blocklist: $CUSTOM_BLOCKLIST"
    echo ""
    
    CUSTOM_BLOCKLIST="$CUSTOM_BLOCKLIST" python3 << 'PYEOF'
import yaml, sys, os

try:
    blocklist = os.environ['CUSTOM_BLOCKLIST']
    with open(blocklist, 'r') as f:
        data = yaml.safe_load(f)
    
    domains = data.get('domains', [])
    enabled = [d for d in domains if d.get('enabled', False)]
    disabled = [d for d in domains if not d.get('enabled', False)]
    
    if enabled:
        print(f"\033[0;32mEnabled ({len(enabled)}):\033[0m")
        for d in enabled:
            cat = d.get('category', 'custom')
            note = d.get('note', '')
            print(f"  ✓  {d['domain']:<30} [{cat:8}] {note}")
    
    if disabled:
        print(f"\n\033[0;33mDisabled ({len(disabled)}):\033[0m")
        for d in disabled:
            cat = d.get('category', 'custom')
            note = d.get('note', '')
            print(f"  ✗  {d['domain']:<30} [{cat:8}] {note}")
    
    if not domains:
        print("  (empty)")
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
PYEOF
}

cmd_status() {
    title "DNS Filter Status"
    echo ""
    
    if is_running; then
        ok "Server: RUNNING"
    else
        err "Server: NOT RUNNING"
        echo "  Start: systemctl start dns-filter"
        return 1
    fi
    
    local stats=$(api_get "/stats")
    
    python3 << PYEOF
import json
try:
    data = json.loads('''$stats''')
    blocked = data.get('blocked_domains', 0)
    s = data.get('stats', {})
    queries = s.get('total_queries', 0)
    blocked_24h = s.get('total_blocked', 0)
    rate = round((blocked_24h / queries * 100), 1) if queries > 0 else 0
    
    print(f"  Blocked Domains: \033[0;32m{blocked:,}\033[0m")
    print(f"  Queries (24h):   {queries:,}")
    print(f"  Blocked (24h):   {blocked_24h:,}")
    print(f"  Block Rate:      {rate}%")
except:
    print("  Stats unavailable")
PYEOF
    
    echo ""
    echo -e "  Installation: ${MAGENTA}$INSTALL_PATH${NC}"
    echo -e "  Dashboard:    ${BLUE}http://127.0.0.1:8080${NC}"
}

cmd_reload() {
    is_running || { err "DNS Filter not running"; return 1; }
    
    info "Reloading custom blocklists..."
    local resp=$(api_post "/blocklist/reload-custom")
    
    if echo "$resp" | grep -q "success"; then
        local count=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('count',0))" 2>/dev/null)
        ok "Reloaded $count domains (cache cleared)"
    else
        err "Reload failed"
    fi
}

cmd_clear() {
    is_running || { err "DNS Filter not running"; return 1; }
    api_post "/system/clear-cache" >/dev/null
    ok "DNS cache cleared"
}

cmd_update() {
    is_running || { err "DNS Filter not running"; return 1; }
    
    info "Updating ALL blocklists (30-60 seconds)..."
    api_post "/blocklist/update" >/dev/null
    ok "Update started (check dashboard for progress)"
}

cmd_info() {
    title "Installation Info"
    echo ""
    echo -e "  Install Path:  ${MAGENTA}$INSTALL_PATH${NC}"
    echo -e "  Config:        $CONFIG_FILE"
    echo -e "  Custom List:   $CUSTOM_BLOCKLIST"
    echo -e "  API:           $API"
    echo -e "  Dashboard:     ${BLUE}http://127.0.0.1:8080${NC}"
    echo ""
    
    if [ -f "$CONFIG_FILE" ]; then
        local dns_port=$(grep "dns_port:" "$CONFIG_FILE" | head -1 | awk '{print $2}')
        local api_port=$(grep "api_port:" "$CONFIG_FILE" | head -1 | awk '{print $2}')
        echo "  DNS Port:      $dns_port"
        echo "  API Port:      $api_port"
    fi
}

cmd_help() {
    cat << 'EOF'

DNS Filter CLI Commands:

  Domain Management:
    block <domain>       Add & enable domain
    unblock <domain>     Disable domain (set enabled: false)
    remove <domain>      Completely remove from list
    enable <domain>      Enable a disabled domain
    disable <domain>     Disable an enabled domain
    whitelist <domain>   Never block (global whitelist)
    test <domain>        Test if domain is blocked
    
  Information:
    status               Server status & stats
    list                 Show all custom domains
    info                 Installation paths
    
  System:
    reload               Reload custom-blocklist.yaml
    update               Update ALL blocklists (slow)
    clear                Clear DNS cache
    
  Navigation:
    help                 This help
    exit / quit          Exit
    
Examples:
  block tiktok.com
  disable tiktok.com
  enable tiktok.com
  test pornhub.com
  list
  status
  
EOF
}

# ── Shell ─────────────────────────────────────────────────────────

print_banner() {
    clear
    echo -e "${GREEN}"
    cat << 'BANNER'
╔═══════════════════════════════════════════════════╗
║                                                   ║
║       DNS FILTER - Interactive CLI v2.1          ║
║                                                   ║
║     Type 'help' for commands, 'exit' to quit     ║
║                                                   ║
╚═══════════════════════════════════════════════════╝
BANNER
    echo -e "${NC}"
    
    echo -e "  Path: ${MAGENTA}$INSTALL_PATH${NC}"
    
    if is_running; then
        ok "Connected to API"
    else
        warn "DNS Filter not responding"
    fi
    echo ""
}

interactive_shell() {
    print_banner
    
    while true; do
        echo -ne "${GREEN}dns-filter>${NC} "
        read -e -r input
        
        [ -n "$input" ] && echo "$input" >> "$HISTFILE" 2>/dev/null
        input=$(echo "$input" | xargs)
        [ -z "$input" ] && continue
        
        local cmd=$(echo "$input" | awk '{print $1}')
        local args=$(echo "$input" | cut -d' ' -f2-)
        [ "$cmd" = "$args" ] && args=""
        
        case "$cmd" in
            block)      cmd_block "$args" ;;
            unblock)    cmd_unblock "$args" ;;
            remove|rm)  cmd_remove "$args" ;;
            enable)     cmd_enable "$args" ;;
            disable)    cmd_disable "$args" ;;
            whitelist)  cmd_whitelist "$args" ;;
            test)       cmd_test "$args" ;;
            list|ls)    cmd_list ;;
            status|stat) cmd_status ;;
            reload)     cmd_reload ;;
            update)     cmd_update ;;
            clear)      cmd_clear ;;
            info)       cmd_info ;;
            help|h|\?)  cmd_help ;;
            exit|quit|q) echo "Goodbye!"; exit 0 ;;
            "") ;;
            *) err "Unknown: $cmd"; echo "Type 'help' for commands" ;;
        esac
        
        echo ""
    done
}

# ── Main ──────────────────────────────────────────────────────────

case "${1:-run}" in
    run) interactive_shell ;;
    *) echo "Usage: dns-cli run"; exit 1 ;;
esac
