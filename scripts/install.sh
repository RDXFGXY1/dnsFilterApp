#!/bin/bash
# DNS Filter - All-in-One Installer
# Handles everything: dependencies, port checks, DNS setup, service installation
# Run: sudo bash install-all.sh

set -e

# ══════════════════════════════════════════════════════════════════
#  Colors & UI
# ══════════════════════════════════════════════════════════════════
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

ok()    { echo -e "${GREEN}✓${NC} $1"; }
err()   { echo -e "${RED}✗${NC} $1"; }
info()  { echo -e "${YELLOW}→${NC} $1"; }
warn()  { echo -e "${YELLOW}!${NC} $1"; }
title() { echo -e "${CYAN}${BOLD}$1${NC}"; }

banner() {
    clear
    echo -e "${GREEN}"
    cat << 'BANNER'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║        DNS CONTENT FILTER - ALL-IN-ONE INSTALLER         ║
║                                                           ║
║              Automatic Setup & Configuration             ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
BANNER
    echo -e "${NC}"
}

# ══════════════════════════════════════════════════════════════════
#  Checks
# ══════════════════════════════════════════════════════════════════

check_root() {
    if [ "$EUID" -ne 0 ]; then
        err "This installer must be run as root"
        echo "  Run: sudo bash $0"
        exit 1
    fi
}

check_os() {
    if [ ! -f /etc/os-release ]; then
        err "Cannot detect OS. Only Linux is supported."
        exit 1
    fi
    
    . /etc/os-release
    ok "Detected: $PRETTY_NAME"
}

check_dependencies() {
    title "Checking dependencies..."
    
    local missing=()
    
    for cmd in curl dig systemctl python3; do
        if ! command -v $cmd >/dev/null 2>&1; then
            missing+=($cmd)
        fi
    done
    
    if [ ${#missing[@]} -gt 0 ]; then
        warn "Missing: ${missing[*]}"
        info "Installing dependencies..."
        
        if command -v apt-get >/dev/null 2>&1; then
            apt-get update -qq
            apt-get install -y curl dnsutils python3 python3-yaml systemd >/dev/null 2>&1
        elif command -v dnf >/dev/null 2>&1; then
            dnf install -y curl bind-utils python3 python3-pyyaml systemd >/dev/null 2>&1
        elif command -v yum >/dev/null 2>&1; then
            yum install -y curl bind-utils python3 python3-pyyaml systemd >/dev/null 2>&1
        elif command -v pacman >/dev/null 2>&1; then
            pacman -S --noconfirm curl dnsutils python python-yaml systemd >/dev/null 2>&1
        else
            err "Cannot install dependencies automatically"
            echo "  Please install: curl, dig, python3, python3-yaml, systemctl"
            exit 1
        fi
        
        ok "Dependencies installed"
    else
        ok "All dependencies present"
    fi
}

check_port_53() {
    title "Checking port 53..."
    
    if lsof -i :53 >/dev/null 2>&1; then
        warn "Port 53 is already in use!"
        echo ""
        lsof -i :53 | head -5
        echo ""
        
        # Identify what's using it
        local process=$(lsof -i :53 -t | head -1)
        local pname=$(ps -p $process -o comm= 2>/dev/null)
        
        if [ "$pname" = "systemd-resolve" ] || [ "$pname" = "systemd-resolved" ]; then
            info "systemd-resolved is using port 53"
            info "Disabling systemd-resolved..."
            
            systemctl stop systemd-resolved >/dev/null 2>&1
            systemctl disable systemd-resolved >/dev/null 2>&1
            
            ok "systemd-resolved disabled"
            
        elif [ "$pname" = "dnsmasq" ]; then
            info "dnsmasq is using port 53"
            info "Stopping dnsmasq..."
            
            systemctl stop dnsmasq >/dev/null 2>&1
            systemctl disable dnsmasq >/dev/null 2>&1
            killall -9 dnsmasq >/dev/null 2>&1
            
            # Also disable dnsmasq in NetworkManager
            if [ -f /etc/NetworkManager/NetworkManager.conf ]; then
                sed -i 's/^dns=dnsmasq/dns=default/' /etc/NetworkManager/NetworkManager.conf
                systemctl restart NetworkManager >/dev/null 2>&1
            fi
            
            ok "dnsmasq stopped and disabled"
            
        else
            err "Unknown process using port 53: $pname (PID: $process)"
            read -p "Kill this process? [y/N] " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill -9 $process
                ok "Process killed"
            else
                err "Cannot continue with port 53 in use"
                exit 1
            fi
        fi
        
        sleep 2
        
        # Verify port is free
        if lsof -i :53 >/dev/null 2>&1; then
            err "Port 53 still in use after cleanup!"
            exit 1
        fi
    fi
    
    ok "Port 53 available"
}

check_binary() {
    title "Checking DNS Filter binary..."
    
    if [ ! -f "./build/dns-filter" ]; then
        err "Binary not found: ./build/dns-filter"
        echo ""
        echo "  Please build first:"
        echo "    make build"
        echo ""
        exit 1
    fi
    
    ok "Binary found"
}

# ══════════════════════════════════════════════════════════════════
#  Installation
# ══════════════════════════════════════════════════════════════════

install_files() {
    title "Installing files to /opt/dns-filter..."
    
    local INSTALL_DIR="/opt/dns-filter"
    
    # Create directories
    mkdir -p $INSTALL_DIR/{configs,data/logs,web/static,web/templates}
    
    # Copy binary
    cp build/dns-filter $INSTALL_DIR/
    chmod +x $INSTALL_DIR/dns-filter
    
    # Copy configs
    cp -r configs/* $INSTALL_DIR/configs/
    
    # Update config.yaml to use /opt paths
    sed -i 's|^\s*custom_path:.*|  custom_path: "/opt/dns-filter/configs/custom*.yaml"|' \
        $INSTALL_DIR/configs/config.yaml 2>/dev/null || true
    
    sed -i 's|^\s*path:.*dns-filter\.db.*|  path: "/opt/dns-filter/data/dns-filter.db"|' \
        $INSTALL_DIR/configs/config.yaml 2>/dev/null || true
    
    sed -i 's|^\s*file:.*dns-filter\.log.*|  file: "/opt/dns-filter/data/logs/dns-filter.log"|' \
        $INSTALL_DIR/configs/config.yaml 2>/dev/null || true
    
    # Copy web files
    cp -r web/static/* $INSTALL_DIR/web/static/
    cp -r web/templates/* $INSTALL_DIR/web/templates/
    
    # Copy CLI
    if [ -f "./dns-cli" ]; then
        cp dns-cli $INSTALL_DIR/dns-cli
        chmod +x $INSTALL_DIR/dns-cli
        ln -sf $INSTALL_DIR/dns-cli /usr/local/bin/dns-cli
        ok "CLI installed: dns-cli"
    elif [ -f "./dns-cli.sh" ]; then
        cp dns-cli.sh $INSTALL_DIR/dns-cli
        chmod +x $INSTALL_DIR/dns-cli
        ln -sf $INSTALL_DIR/dns-cli /usr/local/bin/dns-cli
        ok "CLI installed: dns-cli"
    fi
    
    ok "Files installed to $INSTALL_DIR"
}

install_service() {
    title "Installing systemd service..."
    
    # Create service file
    cat > /etc/systemd/system/dns-filter.service << 'EOF'
[Unit]
Description=DNS Content Filter Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/dns-filter
ExecStart=/opt/dns-filter/dns-filter --config /opt/dns-filter/configs/config.yaml
Restart=always
RestartSec=3

# Security
NoNewPrivileges=false
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable dns-filter >/dev/null 2>&1
    
    ok "Service installed and enabled"
}

# ══════════════════════════════════════════════════════════════════
#  DNS Configuration
# ══════════════════════════════════════════════════════════════════

configure_dns() {
    title "Configuring system DNS..."
    
    # Detect connection manager
    if command -v nmcli >/dev/null 2>&1 && systemctl is-active NetworkManager >/dev/null 2>&1; then
        info "Detected NetworkManager"
        configure_dns_networkmanager
    elif [ -f /etc/resolv.conf ]; then
        info "Using manual resolv.conf"
        configure_dns_resolvconf
    else
        warn "Cannot detect network configuration method"
        return 1
    fi
}

configure_dns_networkmanager() {
    # Get active connection
    local conn=$(nmcli -t -f NAME,DEVICE connection show --active | head -1 | cut -d: -f1)
    
    if [ -z "$conn" ]; then
        warn "No active NetworkManager connection found"
        return 1
    fi
    
    info "Active connection: $conn"
    info "Setting DNS to 127.0.0.1..."
    
    # Set DNS
    nmcli connection modify "$conn" ipv4.dns "127.0.0.1" >/dev/null 2>&1
    nmcli connection modify "$conn" ipv4.ignore-auto-dns yes >/dev/null 2>&1
    
    # Also set IPv6 if enabled
    nmcli connection modify "$conn" ipv6.dns "::1" >/dev/null 2>&1
    nmcli connection modify "$conn" ipv6.ignore-auto-dns yes >/dev/null 2>&1
    
    # Restart connection
    info "Restarting network connection..."
    nmcli connection down "$conn" >/dev/null 2>&1
    sleep 1
    nmcli connection up "$conn" >/dev/null 2>&1
    
    ok "DNS configured via NetworkManager"
}

configure_dns_resolvconf() {
    # Backup original
    if [ ! -f /etc/resolv.conf.backup ]; then
        cp /etc/resolv.conf /etc/resolv.conf.backup
    fi
    
    # Make immutable temporarily
    chattr -i /etc/resolv.conf 2>/dev/null || true
    
    # Write new resolv.conf
    cat > /etc/resolv.conf << EOF
# DNS Filter
nameserver 127.0.0.1
EOF
    
    # Make immutable so it doesn't get overwritten
    chattr +i /etc/resolv.conf 2>/dev/null || true
    
    ok "DNS configured via resolv.conf"
}

# ══════════════════════════════════════════════════════════════════
#  Start & Verify
# ══════════════════════════════════════════════════════════════════

start_service() {
    title "Starting DNS Filter service..."
    
    systemctl start dns-filter
    sleep 3
    
    if systemctl is-active dns-filter >/dev/null 2>&1; then
        ok "Service started successfully"
    else
        err "Service failed to start"
        echo ""
        echo "Check logs:"
        echo "  journalctl -u dns-filter -n 50"
        exit 1
    fi
}

verify_installation() {
    title "Verifying installation..."
    
    # Check service
    if systemctl is-active dns-filter >/dev/null 2>&1; then
        ok "Service is running"
    else
        err "Service is not running"
        return 1
    fi
    
    # Check DNS resolution
    sleep 2
    local test_result=$(dig @127.0.0.1 google.com +short +time=2 2>/dev/null | head -1)
    
    if [ -n "$test_result" ]; then
        ok "DNS resolution working (google.com → $test_result)"
    else
        warn "DNS resolution test failed"
    fi
    
    # Check if blocked domain is blocked
    local blocked_test=$(dig @127.0.0.1 pornhub.com +short +time=2 2>/dev/null)
    
    if [ -z "$blocked_test" ]; then
        ok "Blocking working (pornhub.com blocked)"
    else
        warn "Blocking may not be working yet (blocklists loading...)"
    fi
    
    # Check API
    if curl -s http://127.0.0.1:8080 >/dev/null 2>&1; then
        ok "Web dashboard accessible"
    else
        warn "Dashboard not responding yet"
    fi
}

# ══════════════════════════════════════════════════════════════════
#  Summary
# ══════════════════════════════════════════════════════════════════

show_summary() {
    echo ""
    echo -e "${GREEN}${BOLD}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}${BOLD}║                                                           ║${NC}"
    echo -e "${GREEN}${BOLD}║          INSTALLATION COMPLETE! 🎉                       ║${NC}"
    echo -e "${GREEN}${BOLD}║                                                           ║${NC}"
    echo -e "${GREEN}${BOLD}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    echo -e "${CYAN}Installation Details:${NC}"
    echo -e "  Location:      ${BLUE}/opt/dns-filter${NC}"
    echo -e "  Service:       ${GREEN}dns-filter.service${NC} (enabled & running)"
    echo -e "  DNS Server:    ${GREEN}127.0.0.1:53${NC}"
    echo -e "  Dashboard:     ${BLUE}http://127.0.0.1:8080${NC}"
    echo ""
    
    echo -e "${CYAN}Login Credentials:${NC}"
    echo -e "  Username:      ${GREEN}admin${NC}"
    echo -e "  Password:      ${YELLOW}changeme${NC}"
    echo ""
    
    echo -e "${CYAN}Interactive CLI:${NC}"
    echo -e "  Run:           ${GREEN}dns-cli run${NC}"
    echo -e "  Commands:      block, unblock, test, status, list, etc."
    echo ""
    
    echo -e "${CYAN}Useful Commands:${NC}"
    echo "  systemctl status dns-filter     # Check status"
    echo "  systemctl restart dns-filter    # Restart service"
    echo "  journalctl -u dns-filter -f     # View logs"
    echo "  dns-cli run                     # Open CLI"
    echo ""
    
    echo -e "${CYAN}Test Blocking:${NC}"
    echo "  dig pornhub.com                 # Should return NXDOMAIN"
    echo "  dig google.com                  # Should return IP address"
    echo ""
    
    echo -e "${YELLOW}⚠  IMPORTANT:${NC}"
    echo "  1. Change default password in dashboard"
    echo "  2. Blocklists are downloading in background (30-60 sec)"
    echo "  3. Check dashboard for blocked domain count"
    echo ""
    
    echo -e "${GREEN}Visit dashboard: ${BLUE}http://127.0.0.1:8080${NC}"
    echo ""
}

# ══════════════════════════════════════════════════════════════════
#  Main
# ══════════════════════════════════════════════════════════════════

main() {
    banner
    
    echo -e "${YELLOW}This installer will:${NC}"
    echo "  • Check and install dependencies"
    echo "  • Free port 53 (stop conflicting services)"
    echo "  • Install DNS Filter to /opt/dns-filter"
    echo "  • Configure system DNS to 127.0.0.1"
    echo "  • Start DNS Filter service"
    echo "  • Install interactive CLI tool"
    echo ""
    
    read -p "Continue with installation? [Y/n] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo "Installation cancelled"
        exit 0
    fi
    
    echo ""
    
    # Run all steps
    check_root
    check_os
    check_dependencies
    check_port_53
    check_binary
    
    echo ""
    install_files
    install_service
    
    echo ""
    configure_dns
    
    echo ""
    start_service
    
    echo ""
    verify_installation
    
    echo ""
    show_summary
}

# Run
main "$@"