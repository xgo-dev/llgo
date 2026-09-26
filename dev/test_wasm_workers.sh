#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export LLGO_ROOT="${LLGO_ROOT:-${repo_root}}"
llgo_cmd="${LLGO:-llgo}"
node_cmd="${NODE:-node}"
wasmtime_cmd="${WASMTIME:-wasmtime}"
wasm_tools_cmd="${WASM_TOOLS:-wasm-tools}"
wasm_opt_cmd="${WASMOPT:-wasm-opt}"
worker_fixture="${repo_root}/internal/build/testdata/wasm-workers"
hardening_fixture="${repo_root}/internal/build/testdata/wasm-hardening"
test_fixture="${repo_root}/internal/build/testdata/wasm-test"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-workers.XXXXXX")"
browser_server=

cleanup() {
	if [[ -n "${browser_server}" ]]; then
		kill "${browser_server}" 2>/dev/null || true
		wait "${browser_server}" 2>/dev/null || true
	fi
	rm -rf "${work_dir}"
}
trap cleanup EXIT

run_with_timeout() {
	if command -v timeout >/dev/null 2>&1; then
		timeout 300s "$@"
	else
		"$@"
	fi
}

require_tool() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "$1 is required for the WebAssembly worker acceptance test" >&2
		exit 1
	fi
}

run_emscripten() {
	local target="$1"
	local runner="$2"
	local fixture="$3"
	local expected="$4"
	local name="$5"
	shift 5
	local module="${work_dir}/${name}.mjs"
	local output="${work_dir}/${name}.out"

	LLGO_WASM_WORKERS=2 "${llgo_cmd}" build -target "${target}" -o "${module}" "${fixture}"
	"${wasm_tools_cmd}" validate --features all "${work_dir}/${name}.wasm"
	run_with_timeout env "$@" "${node_cmd}" "${repo_root}/targets/${runner}" "${module}" "${name}" 2>&1 | tee "${output}"
	grep -Fxq "${expected}" "${output}"
}

run_single_hardening() {
	local target="$1"
	local runner="$2"
	local name="$3"
	local module="${work_dir}/${name}.mjs"
	local output="${work_dir}/${name}.out"

	"${llgo_cmd}" build -target "${target}" -o "${module}" "${hardening_fixture}"
	"${wasm_tools_cmd}" validate --features all "${work_dir}/${name}.wasm"
	run_with_timeout env \
		LLGO_WASM_EXPECT_ARG="${name}" \
		LLGO_WASM_BLOCKED_G=1000 \
		"${node_cmd}" "${repo_root}/targets/${runner}" "${module}" "${name}" 2>&1 | tee "${output}"
	grep -Fxq "wasm hardening ok" "${output}"
}

run_worker_llgo_test() {
	local target="$1"
	local name="$2"
	local output="${work_dir}/${name}.out"

	run_with_timeout env LLGO_WASM_WORKERS=2 "${llgo_cmd}" test \
		-target "${target}" -emulator -v -count=1 -timeout=180s \
		"${test_fixture}" 2>&1 | tee "${output}"
	grep -Fq "PASS" "${output}"
}

run_worker_pool_gc_test() {
	local output="${work_dir}/test-pool-gc-emscripten.out"
	run_with_timeout env LLGO_WASM_WORKERS=2 "${llgo_cmd}" test \
		-target emscripten -emulator -v -count=1 -timeout=180s \
		-run '^TestPoolAfterGC$' "${repo_root}/test/std/sync" 2>&1 | tee "${output}"
	grep -Fq "PASS" "${output}"
}

find_browser() {
	if [[ -n "${CHROME:-}" && -x "${CHROME}" ]]; then
		printf '%s\n' "${CHROME}"
		return
	fi
	local candidate
	for candidate in google-chrome google-chrome-stable chromium chromium-browser; do
		if command -v "${candidate}" >/dev/null 2>&1; then
			command -v "${candidate}"
			return
		fi
	done
	for candidate in \
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
		"/Applications/Chromium.app/Contents/MacOS/Chromium"; do
		if [[ -x "${candidate}" ]]; then
			printf '%s\n' "${candidate}"
			return
		fi
	done
	echo "Chrome or Chromium is required for WebAssembly worker browser acceptance" >&2
	return 1
}

run_browser_acceptance() {
	local browser="$1"
	cp "${worker_fixture}/browser.html" "${work_dir}/browser.html"
	"${node_cmd}" "${worker_fixture}/server.mjs" "${work_dir}" 8123 &
	browser_server=$!
	for _ in {1..50}; do
		if curl -fsS http://127.0.0.1:8123/ >/dev/null 2>&1; then
			break
		fi
		sleep 0.1
	done
	curl -fsS http://127.0.0.1:8123/ >/dev/null

	local module
	for module in workers-emscripten.mjs workers-memory64.mjs hardening-workers-emscripten.mjs hardening-workers-memory64.mjs; do
		run_with_timeout "${node_cmd}" "${worker_fixture}/browser-runner.mjs" \
			"${browser}" "http://127.0.0.1:8123/browser.html?module=${module}"
	done

	kill "${browser_server}"
	wait "${browser_server}" 2>/dev/null || true
	browser_server=
}

require_tool "${llgo_cmd}"
require_tool "${node_cmd}"
require_tool "${wasmtime_cmd}"
require_tool "${wasm_tools_cmd}"
require_tool "${wasm_opt_cmd}"
require_tool curl
export WASMOPT="${wasm_opt_cmd}"

# Preserve the single-worker R2 behavior while adding the worker backend.
run_single_hardening emscripten emscripten-runner.mjs hardening-single-emscripten
run_single_hardening emscripten-memory64 emscripten-memory64-runner.mjs hardening-single-memory64
"${llgo_cmd}" build -target wasi -o "${work_dir}/hardening-single-wasi.wasm" "${hardening_fixture}"
"${wasm_tools_cmd}" validate --features all "${work_dir}/hardening-single-wasi.wasm"
run_with_timeout "${wasmtime_cmd}" run -W exceptions=y \
	--env LLGO_WASM_EXPECT_ARG=hardening-single-wasi \
	--env LLGO_WASM_BLOCKED_G=1000 \
	"${work_dir}/hardening-single-wasi.wasm" hardening-single-wasi 2>&1 | tee "${work_dir}/hardening-single-wasi.out"
grep -Fxq "wasm hardening ok" "${work_dir}/hardening-single-wasi.out"

# EC32 and EC64 run the same scheduler, GC, C-boundary, and lifecycle probes.
run_emscripten emscripten emscripten-runner.mjs \
	"${worker_fixture}" "wasm workers ok" workers-emscripten
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs \
	"${worker_fixture}" "wasm workers ok" workers-memory64
run_emscripten emscripten emscripten-runner.mjs \
	"${hardening_fixture}" "wasm hardening ok" hardening-workers-emscripten \
	LLGO_WASM_EXPECT_ARG=hardening-workers-emscripten LLGO_WASM_BLOCKED_G=1000
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs \
	"${hardening_fixture}" "wasm hardening ok" hardening-workers-memory64 \
	LLGO_WASM_EXPECT_ARG=hardening-workers-memory64 LLGO_WASM_BLOCKED_G=1000

# Verify that the public test command selects and executes the worker runtime.
run_worker_llgo_test emscripten test-workers-emscripten
run_worker_llgo_test emscripten-memory64 test-workers-memory64
run_worker_pool_gc_test

run_browser_acceptance "$(find_browser)"

echo "multi-worker WebAssembly runtime checks passed"
