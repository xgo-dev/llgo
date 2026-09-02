#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${1:-.cache/esp-emulator}"
VERSION="0.41.0"
INSTALLER="https://raw.githubusercontent.com/espressif/esp-emulator/v${VERSION}/install.sh"

echo "Installing esp-emu ${VERSION} to ${INSTALL_DIR}/bin"
curl -fsSL "$INSTALLER" | sh -s -- --version "$VERSION" --bin-dir "$INSTALL_DIR/bin"

if [ ! -x "$INSTALL_DIR/bin/esp-emu" ]; then
    echo "Error: esp-emu not found after installation"
    exit 1
fi

"$INSTALL_DIR/bin/esp-emu" --version
