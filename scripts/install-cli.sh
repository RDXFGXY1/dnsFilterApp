#!/bin/bash
# Install DNS Filter CLI to /opt and symlink into /usr/local/bin

set -e

[ "$EUID" -ne 0 ] && echo "Run with sudo" && exit 1

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
INSTALL_PATH="/opt/dns-filter"
CLI_SRC="$SCRIPT_DIR/dns-cli.sh"
CLI_DEST="$INSTALL_PATH/dns-cli.sh"

echo "Installing dns-cli to $INSTALL_PATH"
mkdir -p "$INSTALL_PATH"

if [ ! -f "$CLI_DEST" ]; then
	if [ -f "$CLI_SRC" ]; then
		cp "$CLI_SRC" "$CLI_DEST"
		echo "Copied $CLI_SRC -> $CLI_DEST"
	else
		echo "Source CLI not found: $CLI_SRC"
		echo "Please run the main installer first or place dns-cli.sh next to the project root."
		exit 1
	fi
else
	echo "CLI already present at $CLI_DEST (will ensure up to date)"
	cp "$CLI_SRC" "$CLI_DEST" || true
fi

chmod +x "$CLI_DEST"

ln -sf "$CLI_DEST" /usr/local/bin/dns-filter-cli

echo "✓ Installed: dns-filter-cli -> /usr/local/bin/dns-filter-cli"
echo "  Usage: dns-filter-cli help"
