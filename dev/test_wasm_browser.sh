#!/usr/bin/env bash
set -euo pipefail

llgo_bin=${LLGO:?set LLGO to the llgo executable}
repo_root=${LLGO_ROOT:-$(git rev-parse --show-toplevel)}
browser_bin=${BROWSER_BIN:-}
if [[ -z "$browser_bin" ]]; then
	for candidate in google-chrome-stable google-chrome chromium chromium-browser; do
		if command -v "$candidate" >/dev/null 2>&1; then
			browser_bin=$(command -v "$candidate")
			break
		fi
	done
fi
if [[ -z "$browser_bin" && -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]]; then
	browser_bin="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
fi
if [[ -z "$browser_bin" ]]; then
	echo "a Chrome or Chromium executable is required" >&2
	exit 1
fi

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-browser.XXXXXX")
trap 'rm -rf "$work_dir"' EXIT

cat >"$work_dir/main.go" <<'EOF'
package main

import "syscall/js"

func main() {
	done := make(chan struct{})
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		close(done)
		return nil
	})
	js.Global().Call("setTimeout", callback, 0)
	<-done
	callback.Release()
	js.Global().Get("document").Call("getElementById", "result").Set("textContent", "PASS")
	println("real-browser GoJS callback: PASS")
}
EOF

GOENV=off GOOS=js GOARCH=wasm LLGO_ROOT="$repo_root" "$llgo_bin" build -o "$work_dir/browser.mjs" "$work_dir/main.go"
test -s "$work_dir/browser.mjs"
test -s "$work_dir/browser.wasm"

# Use the existing browser acceptance driver: it gives each invocation an
# isolated Chrome profile, waits for the guest's report in real time, and kills
# the browser process group after a 60-second wall-clock timeout. Chrome's
# --virtual-time-budget alone does not bound --dump-dom on CI hosts.
CHROME="$browser_bin" "${NODE:-node}" "$repo_root/dev/test_wasm_browser.mjs" \
	"$work_dir/browser.mjs" "real-browser GoJS callback: PASS"
