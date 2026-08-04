#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ $# -ne 0 ]]; then
	if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
		echo "usage: dev/test_embed.sh"
		exit 0
	fi
	echo "usage: dev/test_embed.sh" >&2
	exit 2
fi

case "$(uname -s)" in
	Darwin | Linux) ;;
	*)
		echo "error: dev/test_embed.sh supports macOS and Linux; use WSL2 on Windows" >&2
		exit 2
		;;
esac

if [[ "$(uname -s)" == "Darwin" ]]; then
	cache_root="${HOME}/Library/Caches/llgo"
else
	cache_root="${XDG_CACHE_HOME:-$HOME/.cache}/llgo"
fi
qemu_installer="$repo_root/.github/workflows/install-esp-qemu.sh"
qemu_cache_key="$(cksum "$qemu_installer" | awk '{print $1}')"
qemu_dir="$cache_root/esp-qemu/$(uname -m)-$qemu_cache_key"
if [[ ! -x "$qemu_dir/bin/qemu-system-riscv32" || ! -x "$qemu_dir/bin/qemu-system-xtensa" ]]; then
	"$qemu_installer" "$qemu_dir"
fi
export PATH="$qemu_dir/bin:$PATH"

export LLGO_CALLER_PWD="$repo_root"
# shellcheck source=dev/_llgo_setup.sh
source "$repo_root/dev/_llgo_setup.sh"
_llgo_ensure_llgo_cli
llgo_bin_dir="$(dirname "$LLGO_BIN")"
export PATH="$llgo_bin_dir:$PATH"

bash "$repo_root/_demo/embed/test-esp-serial-startup.sh"
