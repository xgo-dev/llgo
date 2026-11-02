#!/usr/bin/env bash

set -euo pipefail

repository="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
archive="$(cygpath -au "$1")"
install_root="$(cygpath -au "$2")"
version="${archive##*/llgo}"
version="${version%%.windows-*}"
architecture="${archive##*.windows-}"
architecture="${architecture%%-*}"
temporary="$(mktemp -d "${TMPDIR:-/tmp}/llgo-msys-path.XXXXXXXX")"
trap 'rm -rf "$temporary"' EXIT

# Reuse the archive and versioned installation already checked by Release CI.
# Only the test HOME is modified; the disposable runner also receives the
# Windows user PATH update from the real PowerShell delegate.
# Run the file so it finds this checkout's install.ps1, not main's companion.
# LLGO_ARCHIVE_PATH selects release installation even inside a source checkout.
env HOME="$temporary" SHELL=/bin/bash \
    LLGO_VERSION="$version" LLGO_ARCH="$architecture" LLGO_INSTALL_ROOT="$install_root" \
    LLGO_ARCHIVE_PATH="$archive" LLGO_INSTALL_DEPS=0 LLGO_UPDATE_PATH=1 \
    bash "$repository/install.sh"

for mode in -ic -lic; do
    # Do not inherit the runner's PATH or use setup-deps' `inherit` setting:
    # the normal MSYS2 launcher uses the minimal Windows PATH.
    # shellcheck disable=SC2016
    env HOME="$temporary" SHELL=/bin/bash MSYS2_PATH_TYPE=minimal \
        PATH=/usr/bin:/bin LLGO_TEST_BIN="$install_root/bin" \
        /usr/bin/bash "$mode" '
            resolved=$(command -v llgo)
            [[ "$resolved" == "$LLGO_TEST_BIN/llgo" || "$resolved" == "$LLGO_TEST_BIN/llgo.exe" ]] || exit 1
            llgo version
        '
    printf 'PASS MSYS2 %s with minimal PATH\n' "$mode"
done
