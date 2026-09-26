#!/bin/bash
# Script to build iwasm with correct options for llgo WASM testing
# This ensures local testing uses the same iwasm configuration as CI

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Match os.UserCacheDir, which backs internal/env.LLGoCacheDir.
source "${SCRIPT_DIR}/llgo_cache_dir.sh"
LLGO_CACHE_DIR="$(llgo_cache_dir)"

IWASM_BIN_DIR="${LLGO_CACHE_DIR}/bin"
WAMR_VERSION="WAMR-2.4.5"

IWASM_NAME="iwasm"
case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*)
        IWASM_NAME="iwasm.exe"
        ;;
esac

# CI keys its cache by this script and patch. Local caches persist across
# checkout updates, so a binary's existence alone cannot prove that it was
# built with the current WASI threads and pthread options.
IWASM_BUILD_ID=$(
    printf '%s\n' "${WAMR_VERSION}" "$(uname -s)" "$(uname -m)" \
        "${LLGO_WINDOWS_ABI:-}" "${MINGW_PREFIX:-}" "${CC:-}" "${CXX:-}" \
        "$(git hash-object "${SCRIPT_DIR}/build_iwasm.sh")" \
        "$(git hash-object "${SCRIPT_DIR}/patches/wamr-2.4.5-mingw.patch")" \
        | git hash-object --stdin
)
IWASM_BUILD_ID_FILE="${IWASM_BIN_DIR}/${IWASM_NAME}.llgo-build-id"
if [ -f "${IWASM_BIN_DIR}/${IWASM_NAME}" ] && \
    [ -f "${IWASM_BUILD_ID_FILE}" ] && \
    [ "$(cat "${IWASM_BUILD_ID_FILE}")" = "${IWASM_BUILD_ID}" ]; then
    echo "Using cached iwasm at ${IWASM_BIN_DIR}/${IWASM_NAME}"
    if [ -n "${GITHUB_PATH:-}" ]; then
        printf '%s\n' "${IWASM_BIN_DIR}" >> "${GITHUB_PATH}"
    fi
    exit 0
fi

echo "Building iwasm for llgo WASM testing..."
echo "Target directory: ${IWASM_BIN_DIR}"

# Create bin directory if it doesn't exist
mkdir -p "${IWASM_BIN_DIR}"

# Create temp directory for building
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT
cd "${TEMP_DIR}"

echo "Cloning wasm-micro-runtime ${WAMR_VERSION}..."
git clone --branch "${WAMR_VERSION}" --depth 1 https://github.com/wasm-micro-runtime/wasm-micro-runtime.git

CMAKE_GENERATOR_ARGS=()
case "$(uname -s)" in
    Darwin)
        PLATFORM="darwin"
        ;;
    Linux)
        PLATFORM="linux"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        PLATFORM="windows"
        IWASM_NAME="iwasm.exe"
        WINDOWS_ABI="${LLGO_WINDOWS_ABI:-}"
        if [ -z "${WINDOWS_ABI}" ]; then
            if [ -n "${MINGW_PREFIX:-}" ]; then
                WINDOWS_ABI="mingw"
            else
                WINDOWS_ABI="msvc"
            fi
        fi
        if [ "${WINDOWS_ABI}" = "mingw" ]; then
            # Keep MinGW on its own GNU ABI toolchain and derive WAMR's target
            # from the active compiler instead of assuming an x86-64 host.
            read -r -a WAMR_CC <<< "${CC:-clang}"
            COMPILER_TARGET=$("${WAMR_CC[@]}" -dumpmachine)
            case "${COMPILER_TARGET%%-*}" in
                x86_64|amd64)
                    WAMR_BUILD_TARGET=X86_64
                    ;;
                i386|i486|i586|i686|x86)
                    WAMR_BUILD_TARGET=X86_32
                    ;;
                aarch64|arm64)
                    WAMR_BUILD_TARGET=AARCH64
                    ;;
                *)
                    echo "Unsupported MinGW WAMR compiler target: ${COMPILER_TARGET}" >&2
                    exit 1
                    ;;
            esac
            CMAKE_GENERATOR_ARGS=(
                -G "MinGW Makefiles"
                -D "CMAKE_C_COMPILER=${CC:-clang}"
                -D "CMAKE_CXX_COMPILER=${CXX:-clang++}"
                -D "CMAKE_C_FLAGS=-fms-extensions -pthread"
                -D "CMAKE_CXX_FLAGS=-fms-extensions -pthread"
                -D "WAMR_BUILD_TARGET=${WAMR_BUILD_TARGET}"
            )
        elif [ "${WINDOWS_ABI}" = "msvc" ]; then
            # The standalone MSVC profile exports nmake and the matching SDK
            # environment without requiring an MSYS2 installation.
            CMAKE_GENERATOR_ARGS=(
                -G "NMake Makefiles"
                -D CMAKE_C_COMPILER=cl
                -D CMAKE_CXX_COMPILER=cl
            )
        else
            echo "Unsupported Windows ABI profile: ${WINDOWS_ABI}" >&2
            exit 1
        fi
        ;;
    *)
        echo "Unsupported platform: $(uname -s)"
        exit 1
        ;;
esac

if [ "${PLATFORM}" = "windows" ] && [ "${WINDOWS_ABI:-}" = "mingw" ]; then
    # WAMR 2.4.5 predates its upstream MinGW source fix. Keep the pinned
    # release and apply that exact backport instead of carrying a local fork.
    git -C wasm-micro-runtime apply \
        "${SCRIPT_DIR}/patches/wamr-2.4.5-mingw.patch"
fi

echo "Building for platform: ${PLATFORM}"

mkdir -p wasm-micro-runtime/product-mini/platforms/${PLATFORM}/build
cd wasm-micro-runtime/product-mini/platforms/${PLATFORM}/build

# The test helper executes Wasm bytecode only, so AOT is unnecessary; LLGo's
# generated modules require reference-types support. WAMR 2.4.5's debug
# interpreter leaves a termination signal after a caught Wasm exception;
# a later branch then aborts LLVM's SjLj path for Go panic/recover.
cmake "${CMAKE_GENERATOR_ARGS[@]}" \
    -D WAMR_BUILD_EXCE_HANDLING=1 \
    -D WAMR_BUILD_AOT=0 \
    -D WAMR_BUILD_FAST_INTERP=0 \
    -D WAMR_BUILD_REF_TYPES=1 \
    -D WAMR_BUILD_SHARED_MEMORY=1 \
    -D WAMR_BUILD_LIB_WASI_THREADS=1 \
    -D WAMR_BUILD_LIB_PTHREAD=1 \
    -D CMAKE_BUILD_TYPE=Debug \
    -D WAMR_BUILD_DEBUG_INTERP=0 \
    ..

echo "Compiling iwasm..."
cmake --build . --parallel "$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"

# Copy iwasm to cache directory
echo "Installing iwasm to ${IWASM_BIN_DIR}..."
cp "${IWASM_NAME}" "${IWASM_BIN_DIR}/"
printf '%s\n' "${IWASM_BUILD_ID}" > "${IWASM_BUILD_ID_FILE}"

if [ -n "${GITHUB_PATH:-}" ]; then
    printf '%s\n' "${IWASM_BIN_DIR}" >> "${GITHUB_PATH}"
fi

echo ""
echo "✓ iwasm successfully built and installed to ${IWASM_BIN_DIR}/${IWASM_NAME}"
echo ""
echo "To use this iwasm, add to your PATH:"
echo "  export PATH=\"${IWASM_BIN_DIR}:\$PATH\""
echo ""
echo "Or run directly:"
echo "  ${IWASM_BIN_DIR}/${IWASM_NAME} --version"
