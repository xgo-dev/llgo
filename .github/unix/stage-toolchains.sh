#!/usr/bin/env bash
# Generate system LLVM target lists without changing the build SDK or runtimes.
set -euo pipefail

source_root=$(cd "${1:-.sysroot}" && pwd -P)
destination=${2:-.release-toolchains}
mkdir -p "$destination"
destination=$(cd "$destination" && pwd -P)
case "$destination/" in
    "$source_root/"*) echo 'Release staging must be outside the build sysroot' >&2; exit 1 ;;
esac
case "$source_root/" in
    "$destination/"*) echo 'Release staging must not contain the build sysroot' >&2; exit 1 ;;
esac

stage=$(mktemp -d "$destination/.staging.XXXXXX")
trap 'rm -rf "$stage"' EXIT
for arch in amd64 arm64; do
    sdk="$source_root/linux/$arch/crosscompile/clang"
    # Fedora's LLVM 22 omits Xtensa. Package compilers use the common targets;
    # the separately staged ESP runtimes retain their full target set.
    config="$stage/system-linux/$arch/include/llvm/Config"
    mkdir -p "$config"
    for definition in Targets AsmParsers AsmPrinters; do
        sed '/^LLVM_[A-Z_]*(Xtensa)$/d' "$sdk/include/llvm/Config/$definition.def" \
            > "$config/$definition.def"
    done
done

rm -rf "${destination:?}/system-linux"
mv "$stage/system-linux" "$destination/system-linux"
