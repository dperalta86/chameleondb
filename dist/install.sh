#!/usr/bin/env sh
set -e

REPO="chameleon-db/chameleondb"
BIN_NAME="chameleon"
INSTALL_DIR="/usr/local/bin"

OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

URL="https://github.com/$REPO/releases/latest/download/chameleon-$OS-$ARCH.tar.gz"

echo "Installing ChameleonDB ($OS/$ARCH)..."
echo "Downloading: $URL"

TMP_DIR="$(mktemp -d)"
curl -fsSL "$URL" -o "$TMP_DIR/chameleon.tar.gz"

tar -xzf "$TMP_DIR/chameleon.tar.gz" -C "$TMP_DIR"

chmod +x "$TMP_DIR/$BIN_NAME"

echo "Installing to $INSTALL_DIR (may require sudo)"
sudo mv "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"

echo "✔ Installation complete"
echo "Run: chameleon --help"
