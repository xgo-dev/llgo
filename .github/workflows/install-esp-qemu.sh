#!/bin/bash
set -euo pipefail

# Installation directory (from argument or default)
INSTALL_DIR="${1:-.cache/qemu}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
EXE_SUFFIX=""

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
  mingw*|msys*|cygwin*)
    # Espressif publishes the Windows emulator as an x86-64 MinGW host
    # binary. This is independent of the ESP firmware target architecture.
    PLATFORM="x86_64-w64-mingw32"
    EXE_SUFFIX=".exe"
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

# SHA-256 values published with the pinned Espressif QEMU release. Keep the
# order aligned with PACKAGES (riscv32, then xtensa).
case "$PLATFORM" in
  aarch64-apple-darwin)
    SHA256S=(
      "234690b6fa7c1d5dfe3dbb2bdd0c2810755e7c98999a9f21c389a6046b7eb76d"
      "aa92e337461d482f5d9f31cd8efc0bd67b3de8fcfcfb567289cb43a59c184651"
    )
    ;;
  aarch64-linux-gnu)
    SHA256S=(
      "f907a54313058f8a9681d2f48257d518950ff98bcd5a319194b4bee7c10cf223"
      "317f6e0fd1dba0886d8110709823d909593ef29438822a14f81ebe19d72ce7cd"
    )
    ;;
  x86_64-apple-darwin)
    SHA256S=(
      "820028ee7cd2dd8fe8cd8ca5519ab6e792d15fea9367c4525cf63c0f707c0b1f"
      "00b9dbc2124cf7633cb86f264fbc524226ad4001bce68bbdba43c9bdc4eb026e"
    )
    ;;
  x86_64-linux-gnu)
    SHA256S=(
      "373b37a68bae3ef441ead24a7bfc950fcbfc274cbdd2b628fc6915f179eb1d8e"
      "588bfaccd0f929650655d10a580f020c6ba9c131712d8fa519280081b8d126eb"
    )
    ;;
  x86_64-w64-mingw32)
    SHA256S=(
      "9474015f24d27acb7516955ec932e5307226bd9d6652cdc870793ed36010ab73"
      "ef550b912726997f3c1ff4a4fb13c1569e2b692efdc5c9f9c3c926a8f7c540fa"
    )
    ;;
esac

echo "Detected platform: $PLATFORM"
echo "Installing to: ${INSTALL_DIR}"

# Download and extract
rm -rf "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

for index in "${!PACKAGES[@]}"; do
  filename="${PACKAGES[$index]}"
  archive="${INSTALL_DIR}/${filename}"
  url="https://github.com/espressif/qemu/releases/download/${RELEASE_TAG}/${filename}"
  echo "Downloading: $url"
  curl -fsSL --retry 5 --retry-all-errors -o "$archive" "$url"
  if command -v sha256sum >/dev/null 2>&1; then
    actual_sha256="$(sha256sum "$archive")"
  else
    actual_sha256="$(shasum -a 256 "$archive")"
  fi
  actual_sha256="${actual_sha256%% *}"
  if [[ "$actual_sha256" != "${SHA256S[$index]}" ]]; then
    echo "SHA-256 mismatch for ${filename}" >&2
    echo "expected: ${SHA256S[$index]}" >&2
    echo "actual:   ${actual_sha256}" >&2
    exit 1
  fi
  tar -xJf "$archive" -C "$INSTALL_DIR" --strip-components=1
  rm -f "$archive"
done

if [[ "$PLATFORM" == "x86_64-w64-mingw32" ]]; then
  # Espressif's Windows archives omit libiconv-2.dll, their only non-system
  # runtime dependency. Install the matching pinned MinGW runtime DLL next to
  # QEMU; this does not make LLGo or its Windows toolchain depend on MSYS2.
  LIBICONV_PACKAGE="mingw-w64-x86_64-libiconv-1.19-1-any.pkg.tar.zst"
  LIBICONV_SHA256="21e334d0911f25de75d3e18e0697648bcecfa9658256d600cad0827d719c2f35"
  LIBICONV_URL="https://repo.msys2.org/mingw/mingw64/${LIBICONV_PACKAGE}"
  libiconv_archive="${INSTALL_DIR}/${LIBICONV_PACKAGE}"
  echo "Downloading: $LIBICONV_URL"
  curl -fsSL --retry 5 --retry-all-errors -o "$libiconv_archive" "$LIBICONV_URL"
  echo "${LIBICONV_SHA256}  ${libiconv_archive}" | sha256sum -c -
  # Git Bash's GNU tar delegates zstd decompression to an optional executable;
  # Windows' built-in bsdtar has native zstd support on every supported runner.
  windows_tar="$(cygpath -u "${SYSTEMROOT}\\System32\\tar.exe")"
  "$windows_tar" -xf "$libiconv_archive" -C "${INSTALL_DIR}/bin" \
    --strip-components=2 mingw64/bin/libiconv-2.dll
  rm -f "$libiconv_archive"
fi

# Verify installation
for exe in qemu-system-riscv32 qemu-system-xtensa; do
  if [ ! -x "${INSTALL_DIR}/bin/${exe}${EXE_SUFFIX}" ]; then
    echo "Error: ${exe} not found after extraction"
    exit 1
  fi
done

if [[ "$PLATFORM" == "x86_64-w64-mingw32" && ! -f "${INSTALL_DIR}/bin/libiconv-2.dll" ]]; then
  echo "Error: libiconv-2.dll not found after extraction"
  exit 1
fi

echo "ESP QEMU (riscv32 + xtensa) installed successfully to: ${INSTALL_DIR}"
