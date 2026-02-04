#!/usr/bin/env sh
set -e

BIN="/usr/local/bin/chameleon"

if [ -f "$BIN" ]; then
  echo "Removing $BIN (may require sudo)"
  sudo rm "$BIN"
  echo "✔ ChameleonDB uninstalled"
else
  echo "ChameleonDB not found"
fi
