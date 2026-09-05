#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
llgo_cmd="${LLGO:-llgo}"
node_cmd="${NODE:-node}"
wasmtime_cmd="${WASMTIME:-wasmtime}"
scheduler_fixture="${repo_root}/internal/build/testdata/wasm-scheduler"
timer_fixture="${repo_root}/internal/build/testdata/wasm-timers"
callback_fixture="${repo_root}/internal/build/testdata/wasm-callback"
gc_fixture="${repo_root}/internal/build/testdata/wasm-gc"
test_fixture="${repo_root}/internal/build/testdata/wasm-test"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-single-worker.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
export LLGO_WASM_TEST_ENV=wasm-env-ok

run_with_timeout() {
	run_with_timeout_limit 180s "$@"
}

run_with_timeout_limit() {
	local limit="$1"
	shift
	if command -v timeout >/dev/null 2>&1; then
		timeout "${limit}" "$@"
	else
		"$@"
	fi
}

expect_failure() {
	local expected="$1"
	shift
	local output exit_code
	set +e
	output="$(run_with_timeout "$@" 2>&1)"
	exit_code=$?
	set -e
	printf '%s\n' "${output}"
	if [[ ${exit_code} -ne 2 ]]; then
		echo "expected exit status 2, got ${exit_code}: $*" >&2
		exit 1
	fi
	grep -Fq "${expected}" <<<"${output}"
}

run_emscripten() {
	local target="$1"
	local runner="$2"
	local fixture="$3"
	local expected="$4"
	local name="$5"
	local module="${work_dir}/${name}.mjs"

	"${llgo_cmd}" build -target "${target}" -o "${module}" "${fixture}"
	wasm-tools validate --features all "${work_dir}/${name}.wasm"
	run_with_timeout "${node_cmd}" "${repo_root}/targets/${runner}" "${module}" 2>&1 | tee "${work_dir}/${name}.out"
	grep -Fq "${expected}" "${work_dir}/${name}.out"
}

run_wasi() {
	local target="$1"
	local fixture="$2"
	local expected="$3"
	local name="$4"
	local module="${work_dir}/${name}.wasm"

	"${llgo_cmd}" build -target "${target}" -o "${module}" "${fixture}"
	wasm-tools validate --features all "${module}"
	run_with_timeout "${wasmtime_cmd}" run -W exceptions=y \
		--env LLGO_WASM_TEST_ENV="${LLGO_WASM_TEST_ENV}" "${module}" 2>&1 | tee "${work_dir}/${name}.out"
	grep -Fq "${expected}" "${work_dir}/${name}.out"
}

run_llgo_test() {
	local target="$1"
	local name="$2"
	local output="${work_dir}/${name}.out"

	# Binaryen's post-Asyncify processing of this standard-library test takes
	# about 165 seconds on a local arm64 host and exceeded 180 seconds on the
	# shared x86-64 runner. Keep execution bounded without treating normal
	# compiler variance as a scheduler failure.
	echo "testing public llgo test command for ${target}"
	run_with_timeout_limit 300s "${llgo_cmd}" test -target "${target}" -emulator \
		-v -count=1 -timeout=30s "${test_fixture}" 2>&1 | tee "${output}"
	grep -Fq "PASS" "${output}"
}

# Canonical C-ecosystem profiles exercise the same scheduler semantics under
# Emscripten wasm32, Emscripten Memory64/LP64, and WASI Preview 1.
run_emscripten emscripten emscripten-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-memory64"
run_wasi wasi "${scheduler_fixture}" "wasm scheduler ok" "scheduler-wasi"

expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${work_dir}/scheduler-wasi.wasm"
expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${work_dir}/scheduler-wasi.wasm"

# Timers share the Go-derived heap but use different host-wait backends.
run_emscripten emscripten emscripten-runner.mjs "${timer_fixture}" "wasm timers ok" "timers-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${timer_fixture}" "wasm timers ok" "timers-memory64"
run_wasi wasi "${timer_fixture}" "wasm timers ok" "timers-wasi"

# R2 enables the non-moving collector by default for each canonical
# single-worker C profile. This fixture covers active and suspended G roots,
# closures/interfaces/aggregates, panic/recover unwinding, pure-Go loop
# safepoints, reclamation, aligned allocation, and memory growth.
run_emscripten emscripten emscripten-runner.mjs "${gc_fixture}" "wasm gc ok" "gc-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${gc_fixture}" "wasm gc ok" "gc-memory64"
run_wasi wasi "${gc_fixture}" "wasm gc ok" "gc-wasi"

# A registered JS callback is a host wake source even when no Go timer exists.
# This catches treating an empty timer heap as an immediate deadlock.
run_emscripten emscripten emscripten-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-memory64"

# Keep the legacy named aliases executable while raw js/wasm remains the
# browser/worker-only compatibility path defined by R0.
run_emscripten wasm emscripten-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasm"
run_wasi wasip1 "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasip1"

# Exercise test-main generation, process exit, verbose output, and host runners
# through the public test command. The JS-specific callback case also verifies
# that host readiness interrupts a longer Go timer wait without re-entering an
# arbitrary parked G.
run_llgo_test emscripten "test-emscripten"
run_llgo_test emscripten-memory64 "test-memory64"
run_llgo_test wasi "test-wasi"

echo "single-worker WebAssembly scheduler and timer checks passed"
