#!/bin/sh
# chr installer
#
# Usage:
#   curl -sSf https://raw.githubusercontent.com/carlosarraes/chr/main/install.sh | sh

set -e

REPO="carlosarraes/chr"
BINARY_NAME="chr"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
GITHUB_LATEST="https://api.github.com/repos/${REPO}/releases/latest"

get_arch() {
    # detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="x86_64" ;;
        aarch64) ARCH="aarch64" ;;
        arm64) ARCH="aarch64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

get_os() {
    # detect os
    OS=$(uname -s)
    case $OS in
        Linux) OS="linux" ;;
        Darwin) OS="darwin" ;;
        *) echo "Unsupported OS: $OS"; exit 1 ;;
    esac
}

download_binary() {
    # get latest release info
    echo "Fetching latest release..."
    VERSION=$(curl -s $GITHUB_LATEST | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4)
    if [ -z "$VERSION" ]; then
        echo "Failed to fetch latest version"
        exit 1
    fi
    
    echo "Latest version: $VERSION"
    
    # create temporary directory
    TMP_DIR=$(mktemp -d)
    echo "Downloading ${BINARY_NAME} ${VERSION}..."
    
    # For now, just download the binary directly (no tar.gz)
    # If we're on a system where the binary is packaged in the future,
    # we can update this logic
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}"
    
    # download the binary directly
    curl -sL "$DOWNLOAD_URL" -o "${TMP_DIR}/${BINARY_NAME}"
    
    # make it executable
    chmod +x "${TMP_DIR}/${BINARY_NAME}"
    
    # install binary
    install -m 755 "${TMP_DIR}/${BINARY_NAME}" "$BIN_DIR"
    
    # cleanup
    rm -rf "$TMP_DIR"
    
    echo "${BINARY_NAME} ${VERSION} installed successfully to $BIN_DIR"
}

# Run the installer
get_arch
get_os
download_binary

echo "Installation complete! Run 'chr --help' to get started." 