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
lifecycle_fixture="${repo_root}/internal/build/testdata/wasm-lifecycle"
test_fixture="${repo_root}/internal/build/testdata/wasm-test"
suite="${1:-all}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-single-worker.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
export LLGO_WASM_TEST_ENV=wasm-env-ok

case "${suite}" in
all | runtime | test-command | gc-heap) ;;
*)
	echo "unknown single-worker WebAssembly suite: ${suite}" >&2
	exit 2
	;;
esac

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

wasi_heap_layout() {
	# Instantiate without starting Go: startup itself needs the GC heap. These
	# linker exports let the test derive the boundary from the actual module.
	"${node_cmd}" - "$1" <<'JS'
const fs = require('node:fs');
const {WASI} = require('node:wasi');
const wasi = new WASI({version: 'preview1', args: ['heap-layout'], env: {}, preopens: {}});
const module = new WebAssembly.Module(fs.readFileSync(process.argv[2]));
const instance = new WebAssembly.Instance(module, {wasi_snapshot_preview1: wasi.wasiImport});
console.log(instance.exports.__global_base.value, instance.exports.__heap_base.value,
  instance.exports.memory.buffer.byteLength);
JS
}

run_wasi_empty_heap() {
	local response="${work_dir}/heap-objects.rsp"
	local module="${work_dir}/gc-wasi-objects.wasm"
	local heap_flags base heap memory aligned_base boundary_heap boundary_memory
	# An object-only user response file must not acquire an implicit 54 MiB
	# heap. No response-file parsing is needed to preserve user memory policy.
	clang --target=wasm32-unknown-unknown -c -x c /dev/null -o "${work_dir}/heap-empty.o"
	printf '"%s"\n' "${work_dir}/heap-empty.o" > "${response}"
	heap_flags="${LDFLAGS:-} -Wl,@${response} -Wl,--export=__heap_base,--export=__global_base"
	# Reuse package archives between these links. Changing only linker options
	# must not require rebuilding the runtime; the rest of the suite keeps its
	# caller-selected cache policy.
	LLGO_BUILD_CACHE=on LDFLAGS="${heap_flags}" \
		run_wasi wasi "${gc_fixture}" "wasm gc ok" "gc-wasi-objects"
	read -r base heap memory < <(wasi_heap_layout "${module}")
	echo "WASI object response: heap base=${heap}, initial memory=${memory}"
	if (( heap > memory || memory - heap >= 65536 )); then
		echo "unexpected initial heap for object response: base=${heap}, memory=${memory}" >&2
		exit 1
	fi

	# Shift static data by its page-rounding slack to create an exact, real
	# __heap_base == memory.size boundary, independent of fixture/code size.
	aligned_base=$((base + memory - heap))
	heap_flags+=" -Wl,--initial-heap=0,--global-base=${aligned_base}"
	LLGO_BUILD_CACHE=on LDFLAGS="${heap_flags}" \
		run_wasi wasi "${gc_fixture}" "wasm gc ok" "gc-wasi-empty"
	read -r base boundary_heap boundary_memory < <(wasi_heap_layout "${work_dir}/gc-wasi-empty.wasm")
	echo "WASI zero heap: heap base=${boundary_heap}, initial memory=${boundary_memory}"
	if (( boundary_heap != boundary_memory )); then
		echo "zero-heap test missed boundary: base=${boundary_heap}, memory=${boundary_memory}" >&2
		exit 1
	fi

	# The same valid module must fail clearly if the host cannot grow its empty
	# heap. Neither an arbitrary trap nor a link failure satisfies this check.
	module="${work_dir}/gc-wasi-empty-limited.wasm"
	LLGO_BUILD_CACHE=on LDFLAGS="${heap_flags} -Wl,--max-memory=${boundary_memory}" \
		"${llgo_cmd}" build -target wasi -o "${module}" "${gc_fixture}"
	wasm-tools validate --features all "${module}"
	expect_failure "gc: invalid heap range" "${wasmtime_cmd}" run -W exceptions=y "${module}"
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

run_llgo_test_compile_only() {
	local target="$1"
	local name="$2"
	local module
	case "${target}" in
	emscripten | emscripten-memory64)
		module="${work_dir}/${name}.mjs"
		;;
	*)
		module="${work_dir}/${name}.wasm"
		;;
	esac

	echo "testing public llgo test -c command for ${target}"
	run_with_timeout_limit 300s "${llgo_cmd}" test -target "${target}" -c \
		-o "${module}" "${test_fixture}"
	case "${module}" in
	*.mjs)
		test -s "${module}"
		wasm-tools validate --features all "${module%.mjs}.wasm"
		;;
	*.wasm)
		wasm-tools validate --features all "${module}"
		;;
	esac
}

if [[ "${suite}" == "all" || "${suite}" == "runtime" ]]; then
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

# Finalizers, cleanups, and weak references share the collector lifecycle but
# have additional ordering, cancellation, and dynamic-call ABI requirements.
run_emscripten emscripten emscripten-runner.mjs "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-memory64"
run_wasi wasi "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-wasi"

# A registered JS callback is a host wake source even when no Go timer exists.
# This catches treating an empty timer heap as an immediate deadlock.
run_emscripten emscripten emscripten-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-emscripten"
run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-memory64"

# Keep the legacy named aliases executable while raw js/wasm remains the
# browser/worker-only compatibility path defined by R0.
run_emscripten wasm emscripten-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasm"
run_wasi wasip1 "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasip1"
fi

if [[ "${suite}" != "test-command" ]]; then
run_wasi_empty_heap
fi

if [[ "${suite}" == "all" || "${suite}" == "test-command" ]]; then
# Exercise test-main generation, process exit, verbose output, and host runners
# through the public test command. The JS-specific callback case also verifies
# that host readiness interrupts a longer Go timer wait without re-entering an
# arbitrary parked G.
run_llgo_test emscripten "test-emscripten"
run_llgo_test emscripten-memory64 "test-memory64"
run_llgo_test wasi "test-wasi"
run_llgo_test_compile_only emscripten "test-compile-only-emscripten"
run_llgo_test_compile_only emscripten-memory64 "test-compile-only-memory64"
run_llgo_test_compile_only wasi "test-compile-only-wasi"
fi

echo "single-worker WebAssembly ${suite} checks passed"
