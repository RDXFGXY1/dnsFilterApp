#!/bin/bash
set -euo pipefail

# Bootstrap installer for DNS Filter
# Clones the repo (or falls back to tarball) and runs `scripts/install.sh`

REPO_URL="https://github.com/RDXFGXY1/dnsFilterApp.git"
TARBALL_URL="https://github.com/RDXFGXY1/dnsFilterApp/archive/refs/heads/main.tar.gz"
BRANCH="main"
INSTALL_DIR="/tmp/dns-filter-setup"

trap 'rm -rf "$INSTALL_DIR"' EXIT

# Ensure running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "Preparing workspace: $INSTALL_DIR"
rm -rf "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

if command -v git >/dev/null 2>&1; then
    echo "Cloning repository (branch: $BRANCH)..."
    git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
else
    echo "git not found, downloading tarball..."
    TMP_TAR="/tmp/dns-filter-setup.tar.gz"
    TMP_DIR="/tmp/dns-filter-tar"
    rm -f "$TMP_TAR"
    rm -rf "$TMP_DIR"
    curl -fsSL "$TARBALL_URL" -o "$TMP_TAR"
    mkdir -p "$TMP_DIR"
    tar -xzf "$TMP_TAR" -C "$TMP_DIR"
    # Move extracted content (first entry) to INSTALL_DIR
    EXTRACTED_DIR=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d | head -n1)
    if [ -z "$EXTRACTED_DIR" ]; then
        echo "Failed to extract repository tarball"
        exit 1
    fi
    mv "$EXTRACTED_DIR"/* "$INSTALL_DIR/"
    rm -rf "$TMP_TAR" "$TMP_DIR"
fi

cd "$INSTALL_DIR"

if [ ! -f scripts/install.sh ]; then
    echo "Installer not found at scripts/install.sh"
    exit 1
fi

# Run project build steps (deps, setup, build) if Make is available
if command -v make >/dev/null 2>&1; then
    echo "Running project build steps: make deps, make setup, make build"
    # tolerate failures but stop on fatal errors due to set -e
    make deps || echo "make deps failed (continuing)"
    make setup || echo "make setup failed (continuing)"
    make build || echo "make build failed (continuing)"
else
    echo "make not found — attempting minimal Go fallback: go mod download && go build"
    if command -v go >/dev/null 2>&1; then
        go mod download || echo "go mod download failed (continuing)"
        go build -o build/dns-filter ./cmd/server || echo "go build failed (continuing)"
    else
        echo "Neither make nor go found — skipping build steps"
    fi
fi

chmod +x scripts/install.sh
echo "Running installer..."
bash scripts/install.sh

echo "Bootstrap finished. Installer cleaned up."
