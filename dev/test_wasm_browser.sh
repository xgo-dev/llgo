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
command -v python3 >/dev/null 2>&1 || {
	echo "python3 is required to serve browser artifacts" >&2
	exit 1
}

work_dir=$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-browser.XXXXXX")
server_pid=
cleanup() {
	if [[ -n "$server_pid" ]]; then
		kill "$server_pid" >/dev/null 2>&1 || true
		wait "$server_pid" >/dev/null 2>&1 || true
	fi
	rm -rf "$work_dir"
}
trap cleanup EXIT

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
}
EOF

cat >"$work_dir/index.html" <<'EOF'
<!doctype html>
<meta charset="utf-8">
<pre id="result">PENDING</pre>
<script type="module">
import Module from "./browser.mjs";

const result = document.getElementById("result");
const fail = error => {
	const status = error?.status;
	if (error?.name === "ExitStatus" && status === 0) return;
	result.textContent = `ERROR: ${error?.stack || error}`;
};
addEventListener("error", event => fail(event.error || event.message));
addEventListener("unhandledrejection", event => {
	if (event.reason?.name === "ExitStatus" && event.reason.status === 0) {
		event.preventDefault();
		return;
	}
	fail(event.reason);
});
setTimeout(() => {
	if (result.textContent === "PENDING") result.textContent = "ERROR: browser timeout";
}, 20000);
Module({
	locateFile: path => new URL(path, import.meta.url).href,
	onExit: status => {
		if (status !== 0) fail(new Error(`guest exit ${status}`));
	},
}).catch(fail);
</script>
EOF

GOENV=off GOOS=js GOARCH=wasm LLGO_ROOT="$repo_root" "$llgo_bin" build -o "$work_dir/browser.mjs" "$work_dir/main.go"
test -s "$work_dir/browser.mjs"
test -s "$work_dir/browser.wasm"

port_file="$work_dir/port"
python3 - "$work_dir" "$port_file" <<'PY' >"$work_dir/server.log" 2>&1 &
import functools
import http.server
import pathlib
import sys

directory, port_file = sys.argv[1:]

class BrowserArtifactHandler(http.server.SimpleHTTPRequestHandler):
	extensions_map = {
		**http.server.SimpleHTTPRequestHandler.extensions_map,
		".mjs": "text/javascript",
		".wasm": "application/wasm",
	}

handler = functools.partial(BrowserArtifactHandler, directory=directory)
server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
pathlib.Path(port_file).write_text(str(server.server_port), encoding="ascii")
server.serve_forever()
PY
server_pid=$!
for _ in $(seq 1 100); do
	[[ -s "$port_file" ]] && break
	sleep 0.05
done
if [[ ! -s "$port_file" ]]; then
	echo "browser artifact server did not start" >&2
	cat "$work_dir/server.log" >&2
	exit 1
fi

port=$(<"$port_file")
"$browser_bin" \
	--headless=new \
	--no-sandbox \
	--disable-gpu \
	--disable-dev-shm-usage \
	--proxy-server=direct:// \
	--proxy-bypass-list='*' \
	--virtual-time-budget=30000 \
	--dump-dom "http://127.0.0.1:$port/index.html" \
	>"$work_dir/dom.html" 2>"$work_dir/browser.log" || {
		cat "$work_dir/browser.log" >&2
		exit 1
	}
if ! grep -q '<pre id="result">PASS</pre>' "$work_dir/dom.html"; then
	cat "$work_dir/dom.html" >&2
	cat "$work_dir/browser.log" >&2
	exit 1
fi

echo "real-browser GoJS callback: PASS"
