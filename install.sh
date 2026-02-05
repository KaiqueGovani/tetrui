#!/usr/bin/env sh
set -e

REPO="${TETRUI_REPO:-KaiqueGovani/tetrui}"
VERSION="${TETRUI_VERSION:-nightly}"
DEST="${DEST:-$HOME/.local/bin/tetrui}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported arch: $ARCH"
    exit 1
    ;;
esac

BIN="tetrui-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BIN}"

mkdir -p "$(dirname "$DEST")"
curl -fsSL "$URL" -o "$DEST"
chmod +x "$DEST"

echo "Installed tetrui to $DEST"
