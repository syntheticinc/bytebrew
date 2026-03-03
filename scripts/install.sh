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

ARCHIVE="bytebrew_${VERSION}_${PLATFORM}.tar.gz"
URL="${BASE_URL}/v${VERSION}/${ARCHIVE}"

echo "Installing ByteBrew CLI v${VERSION} (${PLATFORM})..."
echo ""

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download and extract
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading..."
if ! curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "$URL"; then
  echo "Error: download failed. Check that release v${VERSION} exists for ${PLATFORM}."
  echo "  URL: $URL"
  exit 1
fi

echo "Extracting..."
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

# Install binary
mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "Installed: ${INSTALL_DIR}/${BINARY_NAME}"

# Check PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo ""
    echo "Ready! Run: bytebrew ask \"hello\""
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
    echo "Then run: bytebrew ask \"hello\""
    ;;
esac
