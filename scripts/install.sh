#!/usr/bin/env sh
set -eu

REPO="${REPO:-lihy11/Clash-TUI}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\(.*\)".*/\1/p' | head -n1)"
fi

FILE="clash-tui_${VERSION#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILE"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading $URL"
curl -fL "$URL" -o "$TMP/$FILE"
tar -xzf "$TMP/$FILE" -C "$TMP"

mkdir -p "$INSTALL_DIR"
install "$TMP/clash-tui" "$INSTALL_DIR/clash-tui"

echo "Installed: $INSTALL_DIR/clash-tui"
echo "Run: clash-tui"
