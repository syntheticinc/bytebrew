#!/usr/bin/env bash
set -euo pipefail

# ByteBrew CLI installer for macOS and Linux.
# Usage: curl -fsSL https://bytebrew.ai/releases/install.sh | sh

BASE_URL="https://bytebrew.ai/releases"
INSTALL_DIR="$HOME/.bytebrew/bin"
BINARY_NAME="bytebrew"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  PLATFORM_OS="linux" ;;
  Darwin) PLATFORM_OS="darwin" ;;
  *)
    echo "Error: unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  PLATFORM_ARCH="amd64" ;;
  arm64|aarch64)  PLATFORM_ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH"
    exit 1
    ;;
esac

PLATFORM="${PLATFORM_OS}_${PLATFORM_ARCH}"

# Get latest version
echo "Detecting latest version..."
VERSION=$(curl -fsSL "${BASE_URL}/LATEST")

if [ -z "$VERSION" ]; then
  echo "Error: could not detect latest version. Check ${BASE_URL}/LATEST"
  exit 1
fi

CLI_ARCHIVE="bytebrew_${VERSION}_${PLATFORM}.tar.gz"
CLI_URL="${BASE_URL}/v${VERSION}/${CLI_ARCHIVE}"

SRV_ARCHIVE="bytebrew-srv_${VERSION}_${PLATFORM}.tar.gz"
SRV_URL="${BASE_URL}/v${VERSION}/${SRV_ARCHIVE}"

echo "Installing ByteBrew v${VERSION} (${PLATFORM})..."
echo ""

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download and extract
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# --- CLI ---
echo "Downloading CLI...  ${CLI_ARCHIVE}"
if ! curl -fsSL -o "${TMP_DIR}/${CLI_ARCHIVE}" "$CLI_URL"; then
  echo "Error: CLI download failed. Check that release v${VERSION} exists for ${PLATFORM}."
  echo "  URL: $CLI_URL"
  exit 1
fi

tar -xzf "${TMP_DIR}/${CLI_ARCHIVE}" -C "$TMP_DIR"
mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

# --- Server ---
echo "Downloading Server... ${SRV_ARCHIVE}"
if ! curl -fsSL -o "${TMP_DIR}/${SRV_ARCHIVE}" "$SRV_URL"; then
  echo "Error: Server download failed. Check that release v${VERSION} exists for ${PLATFORM}."
  echo "  URL: $SRV_URL"
  exit 1
fi

tar -xzf "${TMP_DIR}/${SRV_ARCHIVE}" -C "$TMP_DIR"
mv "${TMP_DIR}/bytebrew-srv" "${INSTALL_DIR}/bytebrew-srv"
chmod +x "${INSTALL_DIR}/bytebrew-srv"

echo ""
echo "Installed to ${INSTALL_DIR}"
echo "  bytebrew     (CLI)"
echo "  bytebrew-srv (Server)"

# Check PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo ""
    echo "Ready! Run:"
    echo "  bytebrew login    # authenticate with your account"
    echo "  bytebrew          # start coding"
    ;;
  *)
    echo ""
    echo "Add to PATH (one-time setup):"
    echo ""
    if [ "$PLATFORM_OS" = "darwin" ]; then
      echo "  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.zshrc"
      echo "  source ~/.zshrc"
    else
      echo "  echo 'export PATH=\"\$PATH:${INSTALL_DIR}\"' >> ~/.bashrc"
      echo "  source ~/.bashrc"
    fi
    echo ""
    echo "Then run:"
    echo "  bytebrew login    # authenticate with your account"
    echo "  bytebrew          # start coding"
    ;;
esac
