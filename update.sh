#!/bin/bash
set -euo pipefail

# Update script for DNS Filter
# Usage: curl -fsSL <url>/update.sh | sudo bash

REPO_URL="https://github.com/RDXFGXY1/dnsFilterApp.git"
TARBALL_URL="https://github.com/RDXFGXY1/dnsFilterApp/archive/refs/heads/main.tar.gz"
BRANCH="main"
TMP_DIR="/tmp/dns-filter-update"
INSTALL_DIR="/opt/dns-filter"
SERVICE_NAME="dns-filter"

echo "Starting DNS Filter update..."

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

backup_configs() {
    if [ -d "$INSTALL_DIR/configs" ]; then
        ts=$(date +%Y%m%d%H%M%S)
        echo "Backing up existing configs to $INSTALL_DIR/configs.bak.$ts"
        cp -a "$INSTALL_DIR/configs" "$INSTALL_DIR/configs.bak.$ts"
    fi
}

run_build_steps() {
    # Run build steps as the original sudo user when possible
    if [ -n "${SUDO_USER:-}" ] && [ "${SUDO_USER}" != "root" ]; then
        BUILD_USER="$SUDO_USER"
    else
        BUILD_USER="root"
    fi

    if command -v make >/dev/null 2>&1; then
        echo "Running: make deps && make setup && make build (as $BUILD_USER)"
        if [ "$BUILD_USER" != "root" ]; then
            sudo -u "$BUILD_USER" make deps || true
            sudo -u "$BUILD_USER" make setup || true
            sudo -u "$BUILD_USER" make build || true
        else
            make deps || true
            make setup || true
            make build || true
        fi
    else
        echo "make not found — falling back to go commands"
        if command -v go >/dev/null 2>&1; then
            if [ "$BUILD_USER" != "root" ]; then
                sudo -u "$BUILD_USER" go mod download || true
                sudo -u "$BUILD_USER" go build -o build/dns-filter ./cmd/server || true
            else
                go mod download || true
                go build -o build/dns-filter ./cmd/server || true
            fi
        else
            echo "No build tool found (make/go) — skipping build"
        fi
    fi
}

if [ -d "$INSTALL_DIR/.git" ]; then
    echo "Found existing installation at $INSTALL_DIR — updating via git"
    cd "$INSTALL_DIR"
    git fetch --all --prune
    git reset --hard "origin/$BRANCH"
    git clean -fdx
    # Run build steps in-place
    run_build_steps
    # Ensure binary and cli are executable
    chmod +x "$INSTALL_DIR/dns-filter" || true
    [ -f "$INSTALL_DIR/dns-cli.sh" ] && chmod +x "$INSTALL_DIR/dns-cli.sh" || true
else
    echo "No git repo found at $INSTALL_DIR — performing clean update from remote"
    rm -rf "$TMP_DIR" && mkdir -p "$TMP_DIR"

    if command -v git >/dev/null 2>&1; then
        echo "Cloning repository..."
        git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$TMP_DIR"
    else
        echo "git not available — downloading tarball..."
        curl -fsSL "$TARBALL_URL" -o /tmp/dns-filter-update.tar.gz
        mkdir -p "$TMP_DIR"
        tar -xzf /tmp/dns-filter-update.tar.gz -C "$TMP_DIR"
        # If extraction created a nested folder, move contents up
        firstdir=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d | head -n1 || true)
        if [ -n "$firstdir" ] && [ "$(ls -A "$firstdir")" ]; then
            mv "$firstdir"/* "$TMP_DIR/"
        fi
        rm -f /tmp/dns-filter-update.tar.gz
    fi

    # Build inside tmp dir
    cd "$TMP_DIR"
    run_build_steps

    # Backup existing configs
    if [ -d "$INSTALL_DIR" ]; then
        backup_configs
    else
        mkdir -p "$INSTALL_DIR"
    fi

    # Use rsync if available to copy files while preserving existing configs/data
    if command -v rsync >/dev/null 2>&1; then
        echo "Syncing files to $INSTALL_DIR using rsync (preserving configs and data)"
        rsync -a --delete --exclude 'data' --exclude 'data/*' --exclude 'configs' --exclude 'configs/*' "$TMP_DIR/" "$INSTALL_DIR/"
        # If configs don't exist, copy them
        if [ ! -d "$INSTALL_DIR/configs" ] && [ -d "$TMP_DIR/configs" ]; then
            cp -a "$TMP_DIR/configs" "$INSTALL_DIR/"
        fi
    else
        echo "rsync not found — falling back to cp (will preserve existing configs)"
        # Copy everything except data; preserve configs if present
        tmp_cfg="$INSTALL_DIR/configs.bak.tmp"
        if [ -d "$INSTALL_DIR/configs" ]; then
            mv "$INSTALL_DIR/configs" "$tmp_cfg"
        fi
        cp -a "$TMP_DIR/." "$INSTALL_DIR/"
        if [ -d "$tmp_cfg" ]; then
            mv "$tmp_cfg" "$INSTALL_DIR/configs"
        fi
    fi

    # Ensure binary is in place
    if [ -f "$TMP_DIR/build/dns-filter" ]; then
        cp "$TMP_DIR/build/dns-filter" "$INSTALL_DIR/dns-filter"
        chmod +x "$INSTALL_DIR/dns-filter"
    fi
    if [ -f "$TMP_DIR/dns-cli.sh" ]; then
        cp "$TMP_DIR/dns-cli.sh" "$INSTALL_DIR/dns-cli.sh"
        chmod +x "$INSTALL_DIR/dns-cli.sh"
        ln -sf "$INSTALL_DIR/dns-cli.sh" /usr/local/bin/dns-filter-cli
    fi
fi

echo "Reloading systemd and restarting $SERVICE_NAME service (if present)"
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
    systemctl restart "$SERVICE_NAME" || echo "Failed to restart $SERVICE_NAME (it may not be installed)"
fi

echo "Update complete."
echo "If you made configuration changes, review $INSTALL_DIR/configs/ and restart service if needed."
