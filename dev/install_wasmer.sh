#!/bin/bash
# Install the unmodified, pinned Wasmer CLI used by local and CI WASI tests.
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/llgo_cache_dir.sh"
WASMER_BIN_DIR="$(llgo_cache_dir)/bin"
WASMER_VERSION=7.5.0
WASMER_NAME=wasmer

case "$(uname -s)-$(uname -m)" in
    Darwin-arm64)
        asset=wasmer-darwin-arm64.tar.gz
        digest=fd6526fdce3f8c68a8c3139e9b0e438991d04bafda3d0f091bf98c1600ea0401 ;;
    Linux-x86_64)
        asset=wasmer-linux-amd64.tar.gz
        digest=23f99818deaad93859f8bb8292af81f0c39f25aea19eaccfb72fc7c9e8a3fae9 ;;
    Linux-aarch64|Linux-arm64)
        asset=wasmer-linux-aarch64.tar.gz
        digest=903dadb34ba984d981a9bdd2ed1e456dab862c4489d65d2498ad5e7cf1a1fe87 ;;
    Linux-riscv64)
        asset=wasmer-linux-riscv64.tar.gz
        digest=98fe9ea5e1806cb16659daace881a3f24fae14d0af32a113c4b52b17f28c2261 ;;
    MINGW*-x86_64|MSYS*-x86_64|CYGWIN*-x86_64)
        # This standalone host executable also runs LLGo's MinGW-built guests.
        asset=wasmer-windows-amd64.tar.gz
        digest=05ed663fd0db01d5a690e151c976c384d035a8acac7a7710ffb5fcc9d54c917f
        WASMER_NAME=wasmer.exe ;;
    *)
        echo "Wasmer ${WASMER_VERSION} has no supported prebuilt CLI for $(uname -s)-$(uname -m); install it from source and put wasmer on PATH." >&2
        exit 1 ;;
esac

build_id="${WASMER_VERSION}-${digest}"
id_file="${WASMER_BIN_DIR}/${WASMER_NAME}.llgo-build-id"
if [ -x "${WASMER_BIN_DIR}/${WASMER_NAME}" ] && [ -f "${id_file}" ] && \
    [ "$(cat "${id_file}")" = "${build_id}" ]; then
    echo "Using cached Wasmer at ${WASMER_BIN_DIR}/${WASMER_NAME}"
else
    staging=$(mktemp -d)
    trap 'rm -rf "${staging}"' EXIT
    curl --fail --location --retry 3 \
        "https://github.com/wasmerio/wasmer/releases/download/v${WASMER_VERSION}/${asset}" \
        --output "${staging}/${asset}"
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "${staging}/${asset}")
    else
        actual=$(shasum -a 256 "${staging}/${asset}")
    fi
    if [ "${actual%% *}" != "${digest}" ]; then
        echo "Wasmer archive checksum mismatch: ${asset}" >&2
        exit 1
    fi
    tar -xzf "${staging}/${asset}" -C "${staging}" "bin/${WASMER_NAME}"
    chmod +x "${staging}/bin/${WASMER_NAME}"
    "${staging}/bin/${WASMER_NAME}" --version
    mkdir -p "${WASMER_BIN_DIR}"
    cp "${staging}/bin/${WASMER_NAME}" "${WASMER_BIN_DIR}/${WASMER_NAME}"
    printf '%s\n' "${build_id}" > "${id_file}"
fi

if [ -n "${GITHUB_PATH:-}" ]; then
    printf '%s\n' "${WASMER_BIN_DIR}" >> "${GITHUB_PATH}"
fi
echo "Add ${WASMER_BIN_DIR} to PATH to run WASI programs with Wasmer."
