#!/usr/bin/env bash

set -euo pipefail

readonly LLGO_REPOSITORY="xgo-dev/llgo"
readonly LLGO_GO_VERSION_DEFAULT="1.27.0"

die() {
    printf 'llgo installer: %s\n' "$*" >&2
    exit 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

local_checkout_root() {
    local script="${BASH_SOURCE[0]:-}"
    [[ -f "$script" ]] || return 1
    local root
    root="$(cd "$(dirname "$script")" && pwd)"
    [[ -r "$root/go.mod" ]] || return 1
    local module
    IFS= read -r module <"$root/go.mod" || return 1
    # Git can check out go.mod with CRLF on Windows.
    [[ "${module%$'\r'}" == "module github.com/xgo-dev/llgo" ]] || return 1
    printf '%s\n' "$root"
}

install_local_checkout() {
    local root="$1"
    printf 'Installing llgo from local source...\n'
    (cd "$root" && go install ./cmd/llgo)
    if [[ -n "${GITHUB_ENV:-}" ]]; then
        printf 'LLGO_ROOT=%s\n' "$(windows_native_path "$root")" >>"$GITHUB_ENV"
    fi
    printf 'Local installation complete.\n'
}

normalize_os() {
    case "$1" in
        Darwin | darwin) printf 'darwin\n' ;;
        Linux | linux) printf 'linux\n' ;;
        MINGW*_NT-* | MSYS_NT-* | CYGWIN_NT-*) printf 'windows\n' ;;
        *) return 1 ;;
    esac
}

normalize_arch() {
    case "$1" in
        x86_64 | X86_64 | amd64 | AMD64) printf 'amd64\n' ;;
        arm64 | ARM64 | aarch64 | AARCH64) printf 'arm64\n' ;;
        i386 | i486 | i586 | i686 | x86 | X86) printf '386\n' ;;
        *) return 1 ;;
    esac
}

normalize_version() {
    local version="${1#v}"
    [[ "$version" =~ ^[0-9A-Za-z][0-9A-Za-z.+-]*$ ]] || return 1
    printf '%s\n' "$version"
}

version_at_least() {
    local actual="${1#go}"
    local required="${2#go}"
    awk -v actual="$actual" -v required="$required" 'BEGIN {
        split(actual, a, ".")
        split(required, r, ".")
        for (i = 1; i <= 3; i++) {
            sub(/[^0-9].*$/, "", a[i])
            sub(/[^0-9].*$/, "", r[i])
            av = a[i] + 0
            rv = r[i] + 0
            if (av > rv) exit 0
            if (av < rv) exit 1
        }
        exit 0
    }'
}

sha256_file() {
    if command_exists sha256sum; then
        sha256sum "$1" | awk '{print $1}'
    elif command_exists shasum; then
        shasum -a 256 "$1" | awk '{print $1}'
    elif command_exists openssl; then
        openssl dgst -sha256 "$1" | awk '{print $NF}'
    else
        die "sha256sum, shasum, or openssl is required to verify downloads"
    fi
}

download() {
    local url="$1"
    local output="$2"
    curl --fail --silent --show-error --location --retry 5 --retry-delay 2 \
        --output "$output" "$url"
}

windows_native_path() {
    if command_exists cygpath; then
        cygpath -aw "$1"
    else
        printf '%s\n' "$1"
    fi
}

install_windows_release() {
    local local_archive="$1"
    case "${MSYSTEM:-}" in
        CLANG64 | CLANGARM64) ;;
        *) die "run the MinGW installer from an MSYS2 CLANG64 or CLANGARM64 shell" ;;
    esac

    local shell_root="$LLGO_INSTALL_ROOT"
    if command_exists cygpath; then
        shell_root="$(cygpath -au "$shell_root")"
    fi

    local powershell
    if command_exists powershell.exe; then
        powershell=powershell.exe
    elif command_exists pwsh.exe; then
        powershell=pwsh.exe
    else
        die "PowerShell is required to finish the Windows installation"
    fi

    local temporary
    temporary="$(mktemp -d "${TMPDIR:-/tmp}/llgo-windows-install.XXXXXXXX")"
    local installer="$temporary/install.ps1"
    local source_installer=""
    if [[ -f "${BASH_SOURCE[0]:-}" ]]; then
        source_installer="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/install.ps1"
    fi
    if [[ -n "$source_installer" && -f "$source_installer" ]]; then
        cp "$source_installer" "$installer"
    else
        download "https://raw.githubusercontent.com/$LLGO_REPOSITORY/main/install.ps1" "$installer"
    fi

    export LLGO_ABI=mingw
    export LLGO_VERSION LLGO_INSTALL_DEPS LLGO_UPDATE_PATH LLGO_GO_VERSION LLGO_ARCH
    LLGO_INSTALL_ROOT="$(windows_native_path "$LLGO_INSTALL_ROOT")"
    export LLGO_INSTALL_ROOT
    if [[ -n "$local_archive" ]]; then
        LLGO_ARCHIVE_PATH="$(windows_native_path "$local_archive")"
        export LLGO_ARCHIVE_PATH
    fi

    local native_installer
    native_installer="$(windows_native_path "$installer")"
    local status=0
    "$powershell" -NoProfile -ExecutionPolicy Bypass -File "$native_installer" || status=$?
    rm -rf "$temporary"
    if [[ "$status" == 0 ]]; then
        # MSYS2's default minimal PATH drops Windows user PATH entries. Its own
        # shell profile needs the POSIX path, even after PowerShell updates the
        # Windows PATH. PowerShell already wrote native paths to GITHUB_PATH.
        GITHUB_PATH='' update_shell_path "$shell_root"
    fi
    return "$status"
}

run_as_root() {
    if [[ "$(id -u)" == 0 ]]; then
        "$@"
    elif command_exists sudo; then
        sudo "$@"
    else
        die "installing system dependencies requires root or sudo: $*"
    fi
}

install_macos_dependencies() {
    command_exists xcode-select || die "xcode-select is required on macOS"
    if ! xcode-select -p >/dev/null 2>&1; then
        xcode-select --install || true
        die "finish installing the Xcode Command Line Tools, then run the installer again"
    fi
    command_exists brew || die "Homebrew is required to install LLGo dependencies: https://brew.sh"

    local formula
    local missing=()
    for formula in bdw-gc openssl cjson libffi libuv pkg-config; do
        if ! brew list --versions "$formula" >/dev/null 2>&1; then
            missing+=("$formula")
        fi
    done
    if ((${#missing[@]})); then
        brew install "${missing[@]}"
    fi
}

linux_distribution() {
    [[ -r /etc/os-release ]] || return 1
    # The distribution owns this file and ID/ID_LIKE are shell-compatible.
    # shellcheck disable=SC1091
    . /etc/os-release
    printf '%s %s\n' "${ID:-}" "${ID_LIKE:-}"
}

install_linux_dependencies() {
    local distribution
    distribution="$(linux_distribution || true)"
    case " $distribution " in
        *" debian "* | *" ubuntu "*)
            run_as_root apt-get update
            run_as_root apt-get install -y \
                build-essential ca-certificates curl pkg-config \
                libgc-dev libssl-dev zlib1g-dev libffi-dev libcjson-dev \
                libsqlite3-dev libuv1-dev
            ;;
        *" fedora "* | *" rhel "*)
            run_as_root dnf install -y \
                gcc gcc-c++ make ca-certificates curl pkgconf-pkg-config \
                gc-devel openssl-devel zlib-devel libffi-devel cjson-devel \
                sqlite-devel libuv-devel
            ;;
        *" alpine "*)
            run_as_root apk add \
                build-base ca-certificates curl pkgconf gc-dev openssl-dev \
                zlib-dev libffi-dev cjson-dev sqlite-dev libuv-dev
            ;;
        *" arch "*)
            run_as_root pacman -S --needed --noconfirm \
                base-devel ca-certificates curl pkgconf gc openssl zlib \
                libffi cjson sqlite libuv
            ;;
        *" suse "* | *" opensuse "*)
            run_as_root zypper --non-interactive install \
                gcc gcc-c++ make ca-certificates curl pkg-config \
                libgc-devel libopenssl-devel zlib-devel libffi-devel \
                libcjson-devel sqlite3-devel libuv-devel
            ;;
        *)
            die "unsupported Linux distribution; install a C/C++ toolchain, pkg-config, bdwgc, OpenSSL, zlib, libffi, cJSON, SQLite, and libuv, then rerun with --no-deps"
            ;;
    esac
}

system_go_is_compatible() {
    command_exists go || return 1
    local version
    version="$(go env GOVERSION 2>/dev/null || true)"
    [[ -n "$version" ]] && version_at_least "$version" "$LLGO_GO_VERSION"
}

go_archive_checksum() {
    local metadata="$1"
    local filename="$2"
    awk -v filename="$filename" '
        /"filename"[[:space:]]*:/ {
            candidate = $0
            sub(/^.*"filename"[[:space:]]*:[[:space:]]*"/, "", candidate)
            sub(/".*$/, "", candidate)
            if (found && candidate != filename) exit
            found = (candidate == filename)
        }
        found && /"sha256"[[:space:]]*:/ {
            checksum = $0
            sub(/^.*"sha256"[[:space:]]*:[[:space:]]*"/, "", checksum)
            sub(/".*$/, "", checksum)
            print checksum
            exit
        }
    ' "$metadata"
}

install_official_go() {
    local root="$1"
    local os="$2"
    local arch="$3"
    local filename="go${LLGO_GO_VERSION}.${os}-${arch}.tar.gz"
    local toolchain_root="$root/toolchains/go/$LLGO_GO_VERSION/$os-$arch"

    if [[ ! -x "$toolchain_root/bin/go" ]]; then
        local temporary
        temporary="$(mktemp -d "${TMPDIR:-/tmp}/llgo-go.XXXXXXXX")"
        local metadata="$temporary/releases.json"
        local archive="$temporary/$filename"
        trap 'rm -rf "$temporary"' EXIT

        download "https://go.dev/dl/?mode=json&include=all" "$metadata"
        local expected
        expected="$(go_archive_checksum "$metadata" "$filename")"
        [[ "$expected" =~ ^[0-9a-f]{64}$ ]] || die "Go did not publish a checksum for $filename"

        download "https://go.dev/dl/$filename" "$archive"
        local actual
        actual="$(sha256_file "$archive")"
        [[ "$actual" == "$expected" ]] || die "SHA-256 mismatch for $filename"

        mkdir -p "$(dirname "$toolchain_root")"
        local extracted="$temporary/extracted"
        mkdir -p "$extracted"
        tar -xzf "$archive" -C "$extracted"
        [[ -x "$extracted/go/bin/go" ]] || die "$filename does not contain bin/go"
        mv "$extracted/go" "$toolchain_root"
        trap - EXIT
        rm -rf "$temporary"
    fi

    mkdir -p "$root/bin"
    ln -sfn "../toolchains/go/$LLGO_GO_VERSION/$os-$arch/bin/go" "$root/bin/go"
    ln -sfn "../toolchains/go/$LLGO_GO_VERSION/$os-$arch/bin/gofmt" "$root/bin/gofmt"
    export PATH="$root/bin:$PATH"
}

ensure_go() {
    local root="$1"
    local os="$2"
    local arch="$3"
    if system_go_is_compatible; then
        return
    fi

    if [[ "$os" == darwin ]] && command_exists brew; then
        if brew list --versions go >/dev/null 2>&1; then
            brew upgrade go || true
        else
            brew install go
        fi
        hash -r
        if system_go_is_compatible; then
            return
        fi
    fi

    install_official_go "$root" "$os" "$arch"
    system_go_is_compatible || die "Go $LLGO_GO_VERSION was installed but is not usable"
}

install_dependencies() {
    local root="$1"
    local os="$2"
    local arch="$3"
    export PATH="$root/bin:$PATH"
    local marker="$root/dependencies/unix-v1-$os-$arch-go-$LLGO_GO_VERSION.complete"
    if [[ -f "$marker" ]] && system_go_is_compatible; then
        return
    fi
    case "$os" in
        darwin) install_macos_dependencies ;;
        linux) install_linux_dependencies ;;
    esac
    ensure_go "$root" "$os" "$arch"
    mkdir -p "$(dirname "$marker")"
    : >"$marker"
}

latest_version() {
    local release_url
    release_url="$(curl --fail --silent --show-error --location --head \
        --retry 5 --output /dev/null --write-out '%{url_effective}' \
        "https://github.com/$LLGO_REPOSITORY/releases/latest")" || return 1
    local tag="${release_url##*/}"
    [[ -n "$tag" && "$tag" != latest ]] || return 1
    printf '%s\n' "$tag"
}

install_release() {
    local root="$1"
    local os="$2"
    local arch="$3"
    local requested_version="$4"
    local local_archive="$5"

    local tag="$requested_version"
    if [[ -z "$tag" ]]; then
        tag="$(latest_version)" || die "could not resolve the latest GitHub release"
    fi
    local version
    version="$(normalize_version "$tag")" || die "invalid LLGo version: $tag"
    tag="v$version"

    local platform="$os-$arch"
    local asset="llgo${version}.${platform}.tar.gz"
    local version_root="$root/versions/$version/$platform"
    if [[ ! -x "$version_root/bin/llgo" ]]; then
        local temporary
        temporary="$(mktemp -d "${TMPDIR:-/tmp}/llgo-install.XXXXXXXX")"
        trap 'rm -rf "$temporary"' EXIT
        local archive="$temporary/$asset"

        if [[ -n "$local_archive" ]]; then
            [[ -f "$local_archive" ]] || die "local archive not found: $local_archive"
            cp "$local_archive" "$archive"
        else
            local base="https://github.com/$LLGO_REPOSITORY/releases/download/$tag"
            local checksums="$temporary/checksums.txt"
            download "$base/llgo${version}.checksums.txt" "$checksums"
            download "$base/$asset" "$archive"
            local expected
            expected="$(awk -v asset="$asset" '$2 == asset { print $1; exit }' "$checksums")"
            [[ "$expected" =~ ^[0-9a-f]{64}$ ]] || die "checksum for $asset is missing"
            local actual
            actual="$(sha256_file "$archive")"
            [[ "$actual" == "$expected" ]] || die "SHA-256 mismatch for $asset"
        fi

        local extracted="$temporary/extracted"
        mkdir -p "$extracted"
        tar -xzf "$archive" -C "$extracted"
        [[ -x "$extracted/bin/llgo" ]] || die "$asset does not contain bin/llgo"
        [[ -f "$extracted/runtime/go.mod" ]] || die "$asset does not contain the LLGo runtime"
        mkdir -p "$(dirname "$version_root")"
        mv "$extracted" "$version_root"
        trap - EXIT
        rm -rf "$temporary"
    fi

    mkdir -p "$root/bin"
    if [[ -e "$root/current" && ! -L "$root/current" ]]; then
        die "$root/current exists and is not a symbolic link"
    fi
    local next_current="$root/.current.$$"
    ln -s "versions/$version/$platform" "$next_current"
    rm -f "$root/current"
    mv "$next_current" "$root/current"
    ln -sfn ../current/bin/llgo "$root/bin/llgo"

    printf '%s\n' "$version"
}

update_shell_profile() {
    local profile="$1"
    local line="$2"
    mkdir -p "$(dirname "$profile")"
    if [[ -f "$profile" ]] && grep -Fq '# LLGo installer' "$profile"; then
        if grep -Fqx "$line" "$profile"; then
            return
        fi
        local temporary_profile
        temporary_profile="$(mktemp "${TMPDIR:-/tmp}/llgo-profile.XXXXXXXX")"
        local replaced=0
        local existing
        while IFS= read -r existing || [[ -n "$existing" ]]; do
            if [[ "$existing" == *'# LLGo installer' ]]; then
                if [[ "$replaced" == 0 ]]; then
                    printf '%s\n' "$line" >>"$temporary_profile"
                    replaced=1
                fi
            else
                printf '%s\n' "$existing" >>"$temporary_profile"
            fi
        done <"$profile"
        cat "$temporary_profile" >"$profile"
        rm -f "$temporary_profile"
        printf 'Updated PATH in %s\n' "$profile"
    else
        printf '\n%s\n' "$line" >>"$profile"
        printf 'Updated PATH in %s\n' "$profile"
    fi
}

update_shell_path() {
    local root="$1"
    export PATH="$root/bin:$PATH"
    if [[ -n "${GITHUB_PATH:-}" ]]; then
        printf '%s\n' "$root/bin" >>"$GITHUB_PATH"
    fi
    [[ "$LLGO_UPDATE_PATH" == 1 ]] || return 0

    local shell="${SHELL:-/bin/sh}"
    local profiles=()
    local profile
    case "${shell##*/}" in
        zsh)
            # Both login and non-login interactive zsh read .zshrc.
            profiles=("${ZDOTDIR:-$HOME}/.zshrc")
            profile="${ZDOTDIR:-$HOME}/.zprofile"
            # Refresh entries written by older installers, without otherwise
            # adding a login-only configuration file.
            if [[ -f "$profile" ]] && grep -Fq '# LLGo installer' "$profile"; then
                profiles+=("$profile")
            fi
            ;;
        bash)
            profiles=("$HOME/.bashrc")
            # Bash reads only the first available login file. Do not create a
            # higher-priority file that would hide the user's existing profile.
            for profile in "$HOME/.bash_profile" "$HOME/.bash_login" "$HOME/.profile"; do
                [[ -r "$profile" ]] && break
            done
            profiles+=("$profile")
            ;;
        fish) profiles=("${XDG_CONFIG_HOME:-$HOME/.config}/fish/conf.d/llgo.fish") ;;
        sh | dash) profiles=("$HOME/.profile") ;;
        *)
            printf 'Shell %s is not configured automatically; add %s/bin to PATH.\n' "$shell" "$root"
            return
            ;;
    esac

    local quoted_root="$root/bin"
    if [[ "${shell##*/}" == fish ]]; then
        # Unlike POSIX shells, fish interprets backslashes inside single quotes.
        quoted_root="${quoted_root//\\/\\\\}"
    fi
    local escaped_quote="'\\''"
    quoted_root="'${quoted_root//\'/$escaped_quote}'"
    local line
    if [[ "${shell##*/}" == fish ]]; then
        line="contains -- $quoted_root \$PATH; or set -gx PATH $quoted_root \$PATH # LLGo installer"
    else
        # Login profiles often source the interactive rc file. Keep PATH unique
        # even when both files (or a nested shell) load the installer entry.
        line="case \":\$PATH:\" in *:$quoted_root:*) ;; *) export PATH=$quoted_root:\"\$PATH\" ;; esac # LLGo installer"
    fi
    for profile in "${profiles[@]}"; do
        update_shell_profile "$profile" "$line"
    done
    printf 'Open a new terminal to use the updated PATH; the installer cannot change its parent shell.\n'
}

usage() {
    cat <<'EOF'
Install the latest LLGo release and its native dependencies.

Usage: install.sh [--version VERSION] [--root DIRECTORY] [--no-deps]

Environment variables:
  LLGO_VERSION       release tag or version (default: latest)
  LLGO_INSTALL_ROOT  installation root (default: ~/.llgo)
  LLGO_INSTALL_DEPS  1 to install dependencies, 0 to skip (default: 1)
  LLGO_UPDATE_PATH   1 to update the shell profile, 0 to skip (default: 1)
EOF
}

main() {
    local checkout_root=""
    if [[ -z "${LLGO_ARCHIVE_PATH:-}" ]]; then
        checkout_root="$(local_checkout_root || true)"
    fi
    if [[ -n "$checkout_root" ]]; then
        install_local_checkout "$checkout_root"
        return
    fi

    LLGO_VERSION="${LLGO_VERSION:-}"
    LLGO_INSTALL_ROOT="${LLGO_INSTALL_ROOT:-$HOME/.llgo}"
    LLGO_INSTALL_DEPS="${LLGO_INSTALL_DEPS:-1}"
    LLGO_UPDATE_PATH="${LLGO_UPDATE_PATH:-1}"
    LLGO_GO_VERSION="${LLGO_GO_VERSION:-$LLGO_GO_VERSION_DEFAULT}"
    local local_archive="${LLGO_ARCHIVE_PATH:-}"

    while (($#)); do
        case "$1" in
            --version)
                (($# >= 2)) || die "--version requires a value"
                LLGO_VERSION="$2"
                shift 2
                ;;
            --root)
                (($# >= 2)) || die "--root requires a value"
                LLGO_INSTALL_ROOT="$2"
                shift 2
                ;;
            --no-deps)
                LLGO_INSTALL_DEPS=0
                shift
                ;;
            -h | --help)
                usage
                return
                ;;
            *) die "unknown argument: $1" ;;
        esac
    done

    [[ "$LLGO_INSTALL_DEPS" == 0 || "$LLGO_INSTALL_DEPS" == 1 ]] || die "LLGO_INSTALL_DEPS must be 0 or 1"
    [[ "$LLGO_UPDATE_PATH" == 0 || "$LLGO_UPDATE_PATH" == 1 ]] || die "LLGO_UPDATE_PATH must be 0 or 1"
    local os
    os="$(normalize_os "$(uname -s)")" || die "unsupported operating system: $(uname -s)"
    if [[ "$os" == windows ]]; then
        install_windows_release "$local_archive"
        return
    fi

    [[ -n "$LLGO_INSTALL_ROOT" && "$LLGO_INSTALL_ROOT" != / ]] || die "unsafe installation root"
    case "$LLGO_INSTALL_ROOT" in
        /*) ;;
        *) LLGO_INSTALL_ROOT="$PWD/$LLGO_INSTALL_ROOT" ;;
    esac

    local arch
    arch="$(normalize_arch "$(uname -m)")" || die "unsupported architecture: $(uname -m)"
    [[ "$arch" == amd64 || "$arch" == arm64 ]] || die "LLGo does not publish $os/$arch host archives"

    mkdir -p "$LLGO_INSTALL_ROOT/bin"
    if [[ "$LLGO_INSTALL_DEPS" == 1 ]]; then
        install_dependencies "$LLGO_INSTALL_ROOT" "$os" "$arch"
    fi
    local version
    version="$(install_release "$LLGO_INSTALL_ROOT" "$os" "$arch" "$LLGO_VERSION" "$local_archive")"
    update_shell_path "$LLGO_INSTALL_ROOT"

    "$LLGO_INSTALL_ROOT/bin/llgo" version
    printf 'Installed LLGo %s for %s/%s in %s\n' "$version" "$os" "$arch" "$LLGO_INSTALL_ROOT/current"
    printf 'Stable command: %s/bin/llgo\n' "$LLGO_INSTALL_ROOT"
}

if [[ "${LLGO_INSTALLER_LIBRARY_ONLY:-0}" != 1 ]]; then
    main "$@"
fi
