#!/bin/bash
set -euo pipefail

# Installation directory (from argument or default)
INSTALL_DIR="${1:-.cache/qemu}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture names
case "$ARCH" in
  x86_64|amd64)
    ARCH="x86_64"
    ;;
  aarch64|arm64)
    ARCH="aarch64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Map OS names
case "$OS" in
  darwin)
    PLATFORM="${ARCH}-apple-darwin"
    ;;
  linux)
    PLATFORM="${ARCH}-linux-gnu"
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

RELEASE_TAG="esp-develop-9.2.2-20250817"
VERSION="esp_develop_9.2.2_20250817"
PACKAGES=(
  "qemu-riscv32-softmmu-${VERSION}-${PLATFORM}.tar.xz"
  "qemu-xtensa-softmmu-${VERSION}-${PLATFORM}.tar.xz"
)

echo "Detected platform: $PLATFORM"
echo "Installing to: ${INSTALL_DIR}"

download_and_extract() {
  local url="$1"
  local tmp
  tmp="$(mktemp)"
  trap 'rm -f "$tmp"' RETURN

  curl -fL --retry 5 --retry-all-errors --retry-delay 10 --connect-timeout 30 -o "$tmp" "$url"
  tar -xJ -f "$tmp" -C "$INSTALL_DIR" --strip-components=1
  rm -f "$tmp"
  trap - RETURN
}

# Download and extract
rm -rf "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

for filename in "${PACKAGES[@]}"; do
  url="https://github.com/espressif/qemu/releases/download/${RELEASE_TAG}/${filename}"
  echo "Downloading: $url"
  download_and_extract "$url"
done

# Verify installation
for exe in qemu-system-riscv32 qemu-system-xtensa; do
  if [ ! -x "${INSTALL_DIR}/bin/${exe}" ]; then
    echo "Error: ${exe} not found after extraction"
    exit 1
  fi
done

echo "ESP QEMU (riscv32 + xtensa) installed successfully to: ${INSTALL_DIR}"
