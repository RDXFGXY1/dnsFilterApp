#!/bin/bash
# dns-cli - Interactive DNS Filter Management Shell
# Usage: dns-cli run

VERSION="2.0"
API="http://127.0.0.1:8080/api"
CUSTOM_BLOCKLIST="/opt/dns-filter/configs/custom-blocklist.yaml"
SESSION_COOKIE="session=authenticated"

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly CYAN='\033[0;36m'
readonly BLUE='\033[0;34m'
readonly BOLD='\033[1m'
readonly NC='\033[0m'

# History file
HISTFILE="$HOME/.dns-cli_history"
touch "$HISTFILE"

# Helper functions
ok()    { echo -e "${GREEN}✓${NC} $1"; }
err()   { echo -e "${RED}✗${NC} $1"; }
info()  { echo -e "${YELLOW}→${NC} $1"; }
warn()  { echo -e "${YELLOW}!${NC} $1"; }

is_running() {
    curl -s --max-time 2 "$API/stats" --cookie "$SESSION_COOKIE" >/dev/null 2>&1
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

# Command implementations
cmd_block() {
    local domain="$1"
    
    if [ -z "$domain" ]; then
        err "Usage: block <domain>"
        return 1
    fi
    
    if ! validate_domain "$domain"; then
        err "Invalid domain format: $domain"
        return 1
    fi
    
    if ! is_running; then
        err "DNS Filter not running. Start with: systemctl start dns-filter"
        return 1
    fi
    
    # Add to YAML
    if grep -q "domain: \"$domain\"" "$CUSTOM_BLOCKLIST" 2>/dev/null; then
        info "Domain already in blocklist"
    else
        cat >> "$CUSTOM_BLOCKLIST" << EOF

  - domain: "$domain"
    category: "custom"
    note: "Blocked via CLI"
    enabled: true
EOF
        ok "Added to custom-blocklist.yaml"
    fi
    
    # Reload via API
    local resp=$(api_post "/blocklist/reload-custom")
    if echo "$resp" | grep -q "success"; then
        ok "Blocked: $domain (cache cleared)"
    else
        warn "Added to file but API reload failed. Restart DNS Filter to apply."
    fi
}

cmd_unblock() {
    local domain="$1"
    
    if [ -z "$domain" ]; then
        err "Usage: unblock <domain>"
        return 1
    fi
    
    if ! is_running; then
        err "DNS Filter not running"
        return 1
    fi
    
    api_del "/custom-blocklist/$domain" >/dev/null
    api_post "/system/clear-cache" >/dev/null
    ok "Unblocked: $domain"
}

cmd_whitelist() {
    local domain="$1"
    
    if [ -z "$domain" ]; then
        err "Usage: whitelist <domain>"
        return 1
    fi
    
    if ! validate_domain "$domain"; then
        err "Invalid domain format"
        return 1
    fi
    
    if ! is_running; then
        err "DNS Filter not running"
        return 1
    fi
    
    local resp=$(api_post "/whitelist" "{\"domain\":\"$domain\"}")
    if echo "$resp" | grep -q "success"; then
        ok "Whitelisted: $domain"
    else
        err "Failed to whitelist"
    fi
}

cmd_test() {
    local domain="$1"
    
    if [ -z "$domain" ]; then
        err "Usage: test <domain>"
        return 1
    fi
    
    echo -e "${CYAN}Testing: $domain${NC}"
    
    local result=$(dig @127.0.0.1 "$domain" +short +time=2 2>/dev/null | head -1)
    
    if [ -z "$result" ]; then
        err "BLOCKED - Returns NXDOMAIN"
    else
        ok "ALLOWED - IP: $result"
    fi
}

cmd_list() {
    if [ ! -f "$CUSTOM_BLOCKLIST" ]; then
        err "Custom blocklist not found"
        return 1
    fi
    
    echo -e "${CYAN}Custom Blocklist:${NC}"
    echo ""
    
    python3 << PYEOF
import yaml
import sys

try:
    with open("$CUSTOM_BLOCKLIST", 'r') as f:
        data = yaml.safe_load(f)
    
    domains = data.get('domains', [])
    enabled = [d for d in domains if d.get('enabled', False)]
    disabled = [d for d in domains if not d.get('enabled', False)]
    
    if enabled:
        print(f"\033[0;32mEnabled ({len(enabled)}):\033[0m")
        for d in enabled:
            cat = d.get('category', 'custom')
            note = d.get('note', '')
            print(f"  ✓  {d['domain']:<30} [{cat}] {note}")
    
    if disabled:
        print(f"\n\033[0;33mDisabled ({len(disabled)}):\033[0m")
        for d in disabled:
            cat = d.get('category', 'custom')
            print(f"  ✗  {d['domain']:<30} [{cat}]")
    
    if not domains:
        print("  No custom domains")
        
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
PYEOF
}

cmd_status() {
    echo -e "${CYAN}DNS Filter Status${NC}"
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
import sys

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
    print("  Could not fetch stats")
PYEOF
    
    echo ""
    echo -e "  Dashboard: ${BLUE}http://127.0.0.1:8080${NC}"
}

cmd_reload() {
    if ! is_running; then
        err "DNS Filter not running"
        return 1
    fi
    
    info "Reloading custom blocklists..."
    local resp=$(api_post "/blocklist/reload-custom")
    
    if echo "$resp" | grep -q "success"; then
        local count=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('count',0))" 2>/dev/null)
        ok "Reloaded $count custom domains (cache cleared)"
    else
        err "Reload failed"
    fi
}

cmd_clear() {
    if ! is_running; then
        err "DNS Filter not running"
        return 1
    fi
    
    api_post "/system/clear-cache" >/dev/null
    ok "DNS cache cleared"
}

cmd_help() {
    cat << 'EOF'

DNS Filter CLI Commands:

  Domain Management:
    block <domain>       Block a domain
    unblock <domain>     Unblock a domain
    whitelist <domain>   Never block this domain
    test <domain>        Test if domain is blocked
    
  Information:
    status               Show server status
    list                 Show custom blocked domains
    
  System:
    reload               Reload custom-blocklist.yaml
    clear                Clear DNS cache
    
  Other:
    help                 Show this help
    exit / quit          Exit CLI
    
Examples:
  block tiktok.com
  test pornhub.com
  status
  
EOF
}

# Print banner
print_banner() {
    clear
    echo -e "${GREEN}"
    cat << 'BANNER'
╔═══════════════════════════════════════════════════╗
║                                                   ║
║         DNS FILTER - Interactive CLI v2.0        ║
║                                                   ║
║     Type 'help' for commands, 'exit' to quit     ║
║                                                   ║
╚═══════════════════════════════════════════════════╝
BANNER
    echo -e "${NC}"
    
    # Check connection
    if is_running; then
        ok "Connected to DNS Filter API"
    else
        warn "DNS Filter not responding (is it running?)"
    fi
    echo ""
}

# Interactive shell
interactive_shell() {
    print_banner
    
    # Enable readline if available
    if command -v rlwrap >/dev/null 2>&1; then
        USE_RLWRAP=1
    else
        USE_RLWRAP=0
    fi
    
    while true; do
        # Prompt
        echo -ne "${GREEN}dns-filter>${NC} "
        
        # Read command
        read -e -r input
        
        # Add to history
        [ -n "$input" ] && echo "$input" >> "$HISTFILE"
        
        # Trim whitespace
        input=$(echo "$input" | xargs)
        
        # Skip empty
        [ -z "$input" ] && continue
        
        # Parse command and args
        local cmd=$(echo "$input" | awk '{print $1}')
        local args=$(echo "$input" | cut -d' ' -f2-)
        
        # Handle commands
        case "$cmd" in
            block)
                cmd_block "$args"
                ;;
            unblock)
                cmd_unblock "$args"
                ;;
            whitelist)
                cmd_whitelist "$args"
                ;;
            test)
                cmd_test "$args"
                ;;
            list|ls)
                cmd_list
                ;;
            status|stat)
                cmd_status
                ;;
            reload)
                cmd_reload
                ;;
            clear|cache)
                cmd_clear
                ;;
            help|h|\?)
                cmd_help
                ;;
            exit|quit|q)
                echo "Goodbye!"
                exit 0
                ;;
            "")
                ;;
            *)
                err "Unknown command: $cmd"
                echo "Type 'help' for available commands"
                ;;
        esac
        
        echo ""
    done
}

# Main entry point
case "${1:-run}" in
    run)
        interactive_shell
        ;;
    *)
        echo "DNS Filter CLI"
        echo ""
        echo "Usage:"
        echo "  dns-cli run          Start interactive shell"
        echo ""
        exit 1
        ;;
esac
