#!/bin/sh
set -e

# Detect OS and Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Resolve binary name
case "$OS" in
  darwin)
    BINARY_NAME="kibuild-mcp-darwin-${ARCH}"
    ;;
  linux)
    if [ "$ARCH" = "arm64" ]; then
      echo "Linux ARM64 detected."
      echo "Pre-built Linux ARM64 binaries are not yet provided."
      echo "Please build from source: https://github.com/priyabratasahoo21/kibuild-mcp#build-from-source"
      exit 1
    fi
    BINARY_NAME="kibuild-mcp-linux-${ARCH}"
    ;;
  mingw*|msys*|cygwin*)
    BINARY_NAME="kibuild-mcp-windows-amd64.exe"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

DOWNLOAD_URL="https://github.com/priyabratasahoo21/kibuild-mcp/releases/latest/download/${BINARY_NAME}"
INSTALL_DIR="/usr/local/bin"
TMP_FILE="${TMPDIR:-/tmp}/kibuild-mcp-download"

# Download — prefer curl, fall back to wget
echo "Downloading ${BINARY_NAME}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TMP_FILE" "$DOWNLOAD_URL"
else
  echo "Error: neither curl nor wget found. Please install one and retry."
  exit 1
fi

chmod +x "$TMP_FILE"

# Ensure /usr/local/bin exists
if [ ! -d "$INSTALL_DIR" ]; then
  echo "Creating ${INSTALL_DIR}..."
  sudo mkdir -p "$INSTALL_DIR"
fi

# Move to install dir (sudo if needed)
echo "Installing to ${INSTALL_DIR}/kibuild-mcp..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "${INSTALL_DIR}/kibuild-mcp"
else
  echo "Requesting administrator privileges to install to ${INSTALL_DIR}..."
  sudo mv "$TMP_FILE" "${INSTALL_DIR}/kibuild-mcp"
fi

# macOS: remove Gatekeeper quarantine so MCP clients can spawn the binary
if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/kibuild-mcp" 2>/dev/null || true
  echo "macOS: Gatekeeper quarantine removed."
fi

echo ""
echo "✓ kibuild-mcp installed to ${INSTALL_DIR}/kibuild-mcp"
echo ""
echo "Next: register it in your AI tool's MCP config."
echo "See https://github.com/priyabratasahoo21/kibuild-mcp#setup for instructions."
