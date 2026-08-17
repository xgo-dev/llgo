#!/bin/bash
set -euo pipefail

payload_env=$(mktemp)
archive_file=""
cleanup() {
    rm -f "${payload_env}"
    if [[ -n "${archive_file}" ]]; then
        rm -f "${archive_file}"
    fi
}
trap cleanup EXIT

go run ./internal/llvmpayload/cmd/llvmpayload > "${payload_env}"
# shellcheck disable=SC1090
source "${payload_env}"

get_esp_clang_platform() {
    local platform="$1"
    local os="${platform%-*}"
    local arch="${platform##*-}"
    
    case "${os}" in
        "darwin")
            case "${arch}" in
                "amd64") echo "x86_64-apple-darwin" ;;
                "arm64") echo "aarch64-apple-darwin" ;;
                *) echo "Error: Unsupported darwin architecture: ${arch}" >&2; exit 1 ;;
            esac
            ;;
        "linux")
            case "${arch}" in
                "amd64") echo "x86_64-linux-gnu" ;;
                "arm64") echo "aarch64-linux-gnu" ;;
                *) echo "Error: Unsupported linux architecture: ${arch}" >&2; exit 1 ;;
            esac
            ;;
        *)
            echo "Error: Unsupported OS: ${os}" >&2
            exit 1
            ;;
    esac
}

get_filename() {
    local platform="$1"
    local platform_suffix
    platform_suffix=$(get_esp_clang_platform "${platform}")
    echo "clang-esp-${ESP_CLANG_VERSION}-${platform_suffix}.tar.xz"
}

get_checksum() {
    case "$1" in
        "darwin-amd64") echo "${ESP_CLANG_SHA256_DARWIN_AMD64}" ;;
        "darwin-arm64") echo "${ESP_CLANG_SHA256_DARWIN_ARM64}" ;;
        "linux-amd64") echo "${ESP_CLANG_SHA256_LINUX_AMD64}" ;;
        "linux-arm64") echo "${ESP_CLANG_SHA256_LINUX_ARM64}" ;;
        *) echo "Error: Unsupported checksum platform: $1" >&2; exit 1 ;;
    esac
}

verify_checksum() {
    local filename="$1"
    local expected="$2"
    local actual
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "${filename}" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "${filename}" | awk '{print $1}')
    fi
    if [[ "${actual}" != "${expected}" ]]; then
        echo "Error: checksum mismatch for ${filename}: got ${actual}, want ${expected}" >&2
        exit 1
    fi
}

download_and_extract() {
    local platform="$1"
    local os="${platform%-*}"
    local arch="${platform##*-}"
    local filename
    local checksum
    filename=$(get_filename "${platform}")
    checksum=$(get_checksum "${platform}")
    local download_url="${ESP_CLANG_BASE_URL}/${filename}"
    
    echo "Downloading ESP Clang for ${platform}..."
    echo "  URL: ${download_url}"
    
    archive_file=$(mktemp)
    curl -fsSL "${download_url}" -o "${archive_file}"
    verify_checksum "${archive_file}" "${checksum}"
    mkdir -p ".sysroot/${os}/${arch}/crosscompile/clang"
    tar -xJf "${archive_file}" -C ".sysroot/${os}/${arch}/crosscompile/clang" --strip-components=1
    rm -f "${archive_file}"
    archive_file=""
    
    if [[ ! -f ".sysroot/${os}/${arch}/crosscompile/clang/bin/clang++" ]]; then
        echo "Error: clang++ not found in ${platform} toolchain"
        exit 1
    fi
    
    echo "${platform} ESP Clang ready in .sysroot/${os}/${arch}/crosscompile/clang"
}

echo "Downloading ESP Clang toolchain version ${ESP_CLANG_VERSION}..."

if [[ -n "${GITHUB_ENV:-}" ]]; then
    echo "LLGO_LLVM_MAJOR=${ESP_CLANG_LLVM_MAJOR}" >> "${GITHUB_ENV}"
fi

for platform in "darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64"; do
    download_and_extract "${platform}"
done

echo "ESP Clang toolchain completed successfully!"
