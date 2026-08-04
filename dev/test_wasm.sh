#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ $# -ne 0 ]]; then
	if [[ $# -eq 1 && ("$1" == "-h" || "$1" == "--help") ]]; then
		echo "usage: dev/test_wasm.sh"
		exit 0
	fi
	echo "usage: dev/test_wasm.sh" >&2
	exit 2
fi

case "$(uname -s)" in
	Darwin | Linux) ;;
	*)
		echo "error: dev/test_wasm.sh supports macOS and Linux; use WSL2 on Windows" >&2
		exit 2
		;;
esac

if [[ "$(uname -s)" == "Darwin" ]]; then
	iwasm_bin="${HOME}/Library/Caches/llgo/bin/iwasm"
else
	iwasm_bin="${XDG_CACHE_HOME:-$HOME/.cache}/llgo/bin/iwasm"
fi
if [[ ! -x "$iwasm_bin" ]]; then
	"$repo_root/dev/build_iwasm.sh"
fi
if [[ ! -x "$iwasm_bin" ]]; then
	echo "error: iwasm was not installed at $iwasm_bin" >&2
	exit 1
fi

tmp_dir="$(mktemp -d)"
cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup EXIT

(
	cd "$repo_root/_demo/c"
	"$repo_root/dev/llgo_wasm.sh" build -o "$tmp_dir/hello" -tags=nogc ./helloc
)

wasm_file="$tmp_dir/hello.wasm"
if [[ ! -f "$wasm_file" && -f "$tmp_dir/hello" ]]; then
	wasm_file="$tmp_dir/hello"
fi
if [[ ! -f "$wasm_file" ]]; then
	echo "error: wasm output not found under $tmp_dir" >&2
	exit 1
fi

"$iwasm_bin" --stack-size=819200000 --heap-size=800000000 "$wasm_file"
