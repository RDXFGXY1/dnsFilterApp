#!/bin/bash

# DNS Filter Installation Script
# This script installs DNS Filter as a system service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root (use sudo)${NC}"
    exit 1
fi

echo -e "${GREEN}╔═══════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   DNS Content Filter - Installer     ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════╝${NC}"
echo ""

# Installation directory
INSTALL_DIR="/opt/dns-filter"

# Create installation directory
echo -e "${YELLOW}Creating installation directory...${NC}"
mkdir -p $INSTALL_DIR
mkdir -p $INSTALL_DIR/configs
mkdir -p $INSTALL_DIR/data/logs
mkdir -p $INSTALL_DIR/web

# Copy files
echo -e "${YELLOW}Copying files...${NC}"
cp -r build/dns-filter $INSTALL_DIR/
cp -r configs/* $INSTALL_DIR/configs/
cp -r web/* $INSTALL_DIR/web/

# Install CLI script to the installation directory and symlink into PATH
echo -e "${YELLOW}Installing CLI script...${NC}"
cp dns-cli.sh $INSTALL_DIR/dns-cli.sh
chmod +x $INSTALL_DIR/dns-cli.sh
ln -sf $INSTALL_DIR/dns-cli.sh /usr/local/bin/dns-filter-cli

# Set permissions
chmod +x $INSTALL_DIR/dns-filter

# Install systemd service
echo -e "${YELLOW}Installing systemd service...${NC}"
cp scripts/dns-filter.service /etc/systemd/system/
systemctl daemon-reload

# Enable service
echo -e "${YELLOW}Enabling service...${NC}"
systemctl enable dns-filter

echo ""
echo -e "${GREEN}✓ Installation complete!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Edit configuration: $INSTALL_DIR/configs/config.yaml"
echo "2. Start the service: sudo systemctl start dns-filter"
echo "3. Check status: sudo systemctl status dns-filter"
echo "4. Set your system DNS to 127.0.0.1"
echo "5. Access dashboard: http://localhost:8080"
echo "6. Run the CLI from anywhere: dns-filter-cli run"
echo ""
echo -e "${YELLOW}Default login:${NC}"
echo "  Username: admin"
echo "  Password: changeme"
echo ""
echo -e "${RED}⚠ Remember to change the default password!${NC}"
