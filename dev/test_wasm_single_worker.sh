#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/wasm_ci_report.sh"
llgo_cmd="${LLGO:-llgo}"
node_cmd="${NODE:-node}"
wasmtime_cmd="${WASMTIME:-wasmtime}"
scheduler_fixture="${repo_root}/internal/build/testdata/wasm-scheduler"
timer_fixture="${repo_root}/internal/build/testdata/wasm-timers"
callback_fixture="${repo_root}/internal/build/testdata/wasm-callback"
gc_fixture="${repo_root}/internal/build/testdata/wasm-gc"
lifecycle_fixture="${repo_root}/internal/build/testdata/wasm-lifecycle"
test_fixture="${repo_root}/internal/build/testdata/wasm-test"
runner_test_fixture="${repo_root}/internal/build/testdata/wasm-runner-test"
runner_run_fixture="${repo_root}/internal/build/testdata/wasm-runner-run"
secondary_test_fixture="${repo_root}/internal/build/testdata/wasm-test-secondary"
suite="${1:-all}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-single-worker.XXXXXX")"
report_file="${work_dir}/coverage.tsv"
wasm_ci_report_init "${report_file}"

finish() {
	local status=$?
	trap - EXIT
	set +e
	wasm_ci_publish_report "${report_file}" "${status}"
	rm -rf "${work_dir}"
	exit "${status}"
}
trap finish EXIT
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

assert_no_implicit_wasm_artifacts() {
	local temp_dir="$1"
	local artifact
	artifact="$(find "${temp_dir}" -type f \( -name '*.mjs' -o -name '*.wasm' \) -print -quit)"
	if [[ -n "${artifact}" ]]; then
		echo "implicit WebAssembly artifact was not removed: ${artifact}" >&2
		exit 1
	fi
}

expect_llgo_runner_failure() {
	local target="$1"
	local profile="$2"
	local runner="$3"
	local fixture="$4"
	local output exit_code
	local temp_dir="${work_dir}/runner-failure-${target}-tmp"
	mkdir -p "${temp_dir}"

	set +e
	output="$(run_with_timeout env TMPDIR="${temp_dir}" LLGO_WASM_SCHEDULER_DEADLOCK=1 \
		"${llgo_cmd}" run -target "${target}" -emulator "${fixture}" 2>&1)"
	exit_code=$?
	set -e
	printf '%s\n' "${output}"
	if [[ ${exit_code} -ne 1 ]]; then
		echo "expected llgo runner failure status 1, got ${exit_code}" >&2
		exit 1
	fi
	for expected in \
		"phase=run" \
		"target=${target}" \
		"profile=${profile}" \
		'artifact="' \
		"runner=\"${runner}\"" \
		'package="github.com/xgo-dev/llgo/internal/build/testdata/wasm-scheduler"' \
		"status=exit" \
		"exit_code=2"; do
		grep -Fq "${expected}" <<<"${output}"
	done
	assert_no_implicit_wasm_artifacts "${temp_dir}"
}

expect_llgo_runner_timeout() {
	local target="$1"
	local profile="$2"
	local runner="$3"
	local fixture="$4"
	local output exit_code
	local temp_dir="${work_dir}/runner-timeout-${target}-tmp"
	mkdir -p "${temp_dir}"

	set +e
	output="$(run_with_timeout env TMPDIR="${temp_dir}" LLGO_WASM_SCHEDULER_HANG=1 \
		"${llgo_cmd}" run -timeout=1s -target "${target}" -emulator "${fixture}" 2>&1)"
	exit_code=$?
	set -e
	printf '%s\n' "${output}"
	if [[ ${exit_code} -ne 1 ]]; then
		echo "expected llgo runner timeout status 1, got ${exit_code}" >&2
		exit 1
	fi
	for expected in \
		"phase=run" \
		"target=${target}" \
		"profile=${profile}" \
		'artifact="' \
		"runner=\"${runner}\"" \
		'package="github.com/xgo-dev/llgo/internal/build/testdata/wasm-scheduler"' \
		"status=timeout" \
		"timeout=1s"; do
		grep -Fq "${expected}" <<<"${output}"
	done
	assert_no_implicit_wasm_artifacts "${temp_dir}"
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

run_browser() {
	local module="$1"
	local expected="$2"
	run_with_timeout_limit 90s "${node_cmd}" "${repo_root}/dev/test_wasm_browser.mjs" \
		"${module}" "${expected}"
}

expect_browser_load_failure() {
	local status=0
	run_browser "${work_dir}/missing-browser-module.mjs" "must not pass" \
		> "${work_dir}/browser-failure.out" 2>&1 || status=$?
	cat "${work_dir}/browser-failure.out"
	if [[ ${status} -ne 1 ]] || ! grep -Fq "WebAssembly browser module failed:" "${work_dir}/browser-failure.out"; then
		echo "expected an explicit browser page failure, got exit ${status}" >&2
		exit 1
	fi
}

run_host_call_boundaries() {
	local module="$1"
	local mode operation status expected marker
	for mode in return throw exit-0 exit-7; do
		for operation in Call Invoke New; do
			expected=0
			marker="wasm host call boundary ok"
			if [[ "${mode}" == exit-* ]]; then
				expected="${mode#exit-}"
				marker="wasm host exit reached"
			fi
			status=0
			run_with_timeout "${node_cmd}" "${repo_root}/dev/test_wasm_js_boundary.mjs" \
				"${module}" "${mode}" "${operation}" > "${work_dir}/host-call.out" 2>&1 || status=$?
			cat "${work_dir}/host-call.out"
			if [[ ${status} -ne ${expected} ]]; then
				echo "${mode}/${operation}: expected exit ${expected}, got ${status}" >&2
				exit 1
			fi
			grep -Fq "${marker}" "${work_dir}/host-call.out"
			grep -Fq "wasm host boundary runner ok" "${work_dir}/host-call.out"
			if grep -Eq '^(panic:|fatal error:)' "${work_dir}/host-call.out"; then
				exit 1
			fi
		done
	done
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
	local wasm_page_size=65536
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
	if (( heap > memory || memory - heap >= wasm_page_size )); then
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

run_llgo_run() {
	local target="$1"
	local fixture="$2"
	local expected="$3"
	local name="$4"
	local output="${work_dir}/${name}.out"
	local temp_dir="${work_dir}/${name}-tmp"
	mkdir -p "${temp_dir}"

	# Exercise the public command and its target-owned runner. Other fixtures in
	# this script retain explicit artifacts for wasm-tools validation, while this
	# path verifies that users do not need to assemble Node or Wasmtime commands.
	echo "testing public llgo run command for ${target}"
	run_with_timeout env TMPDIR="${temp_dir}" "${llgo_cmd}" run -target "${target}" -emulator "${fixture}" 2>&1 | tee "${output}"
	grep -Fq "${expected}" "${output}"
	assert_no_implicit_wasm_artifacts "${temp_dir}"
}

run_llgo_test() {
	local target="$1"
	local name="$2"
	local output="${work_dir}/${name}.out"
	local temp_dir="${work_dir}/${name}-tmp"
	mkdir -p "${temp_dir}"

	# This command builds and runs two packages. CI spent 258 seconds on the
	# first WASI package alone, then hit the old 300-second aggregate limit
	# while building the second. Budget 300 seconds per package; each binary
	# has a finite 90-second test deadline and bounded host runner. The EC32
	# reflection tests exceeded 30 seconds as a complete package on CI.
	echo "testing public llgo test command for ${target}"
	run_with_timeout_limit 600s env TMPDIR="${temp_dir}" "${llgo_cmd}" test -target "${target}" -emulator \
		-v -count=1 -timeout=90s "${test_fixture}" "${secondary_test_fixture}" 2>&1 | tee "${output}"
	grep -Fq "PASS" "${output}"
	grep -Fq "TestScheduler" "${output}"
	grep -Fq "wasm secondary package ok" "${output}"
	assert_no_implicit_wasm_artifacts "${temp_dir}"
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
LLGO="${llgo_cmd}" NODE="${node_cmd}" bash "${repo_root}/dev/test_wasm_stack_bounds.sh"
# The runtime is a nested Go module and is not covered by root-level go test.
# Exercise the collector's allocation-free interval helper on the host before
# the target fixtures check marking, reclamation, and finalizer integration.
go -C "${repo_root}/runtime" test -count=1 -cover ./internal/runtime/tinygogc
fi

run_llgo_go_profile_test() {
	local goos="$1"
	local name="$2"
	local pattern="${3:-}"
	local fixture="${4:-${test_fixture}}"
	local output="${work_dir}/${name}.out"
	# The GOOS=js route runs the same reflection-heavy package as EC32.
	# Keep its package deadline aligned with the public target route above.
	local test_args=(-v -count=1 -timeout=90s)
	if [[ -n "${pattern}" ]]; then
		test_args+=(-run "${pattern}")
	fi

	echo "testing public llgo test command for GOOS=${goos} GOARCH=wasm"
	run_with_timeout_limit 300s env GOOS="${goos}" GOARCH=wasm \
		"${llgo_cmd}" test "${test_args[@]}" "${fixture}" 2>&1 | tee "${output}"
	grep -Fq "PASS" "${output}"
}

run_llgo_go_profile_run() {
	local goos="$1"
	local name="$2"
	local output="${work_dir}/${name}.out"

	echo "testing public llgo run command for GOOS=${goos} GOARCH=wasm"
	run_with_timeout_limit 300s env GOOS="${goos}" GOARCH=wasm \
		"${llgo_cmd}" run "${runner_run_fixture}" 2>&1 | tee "${output}"
	grep -Fq "raw wasm run ok" "${output}"
}

if [[ "${suite}" == "all" || "${suite}" == "runtime" ]]; then
# Check finalizer registry scaling and queue removal without timing or
# conservative-root assumptions, alongside the real Wasm lifecycle fixtures.
go -C "${repo_root}" test ./internal/build -run '^TestWasmFinalizerCandidates$' -count=1

# Canonical hosted targets exercise the same scheduler semantics under J32
# Emscripten, J64 Emscripten Memory64, and W32 WASI Preview 1.
wasm_ci_run_case EC32/emscripten scheduler 1 0 0 0 0 \
	run_emscripten emscripten emscripten-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 scheduler 1 0 0 0 0 \
	run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${scheduler_fixture}" "wasm scheduler ok" "scheduler-memory64"
wasm_ci_run_case WC32/wasi scheduler 1 0 0 0 0 \
	run_wasi wasi "${scheduler_fixture}" "wasm scheduler ok" "scheduler-wasi"

wasm_ci_run_case EC32/emscripten scheduler-deadlock 1 1 0 0 0 \
	expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
wasm_ci_run_case EC32/emscripten scheduler-main-goexit 1 1 0 0 0 \
	expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
wasm_ci_run_case EC64/emscripten-memory64 scheduler-deadlock 1 1 0 0 0 \
	expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
wasm_ci_run_case EC64/emscripten-memory64 scheduler-main-goexit 1 1 0 0 0 \
	expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
wasm_ci_run_case WC32/wasi scheduler-deadlock 1 1 0 0 0 \
	expect_failure "fatal error: all goroutines are asleep - deadlock!" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_DEADLOCK=1 "${work_dir}/scheduler-wasi.wasm"
wasm_ci_run_case WC32/wasi scheduler-main-goexit 1 1 0 0 0 \
	expect_failure "fatal error: no goroutines (main called runtime.Goexit) - deadlock!" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_MAIN_GOEXIT=1 "${work_dir}/scheduler-wasi.wasm"

# Reuse the scheduler artifacts to verify that unrecovered panics retain Go
# function names under every canonical host provider.
expect_failure "main.panicTracebackCaller" \
	env LLGO_WASM_SCHEDULER_PANIC_TRACEBACK=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
expect_failure "main.panicTracebackCaller" \
	env LLGO_WASM_SCHEDULER_PANIC_TRACEBACK=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
expect_failure "main.panicTracebackCaller" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_PANIC_TRACEBACK=1 "${work_dir}/scheduler-wasi.wasm"

# A recovered panic rethrown from nested deferred activations keeps the
# original panic site, matching Go's same-value repanic traceback semantics.
expect_failure "main.repanicTracebackOrigin" \
	env LLGO_WASM_SCHEDULER_REPANIC_TRACEBACK=1 "${node_cmd}" "${repo_root}/targets/emscripten-runner.mjs" "${work_dir}/scheduler-emscripten.mjs"
expect_failure "main.repanicTracebackOrigin" \
	env LLGO_WASM_SCHEDULER_REPANIC_TRACEBACK=1 "${node_cmd}" "${repo_root}/targets/emscripten-memory64-runner.mjs" "${work_dir}/scheduler-memory64.mjs"
expect_failure "main.repanicTracebackOrigin" \
	"${wasmtime_cmd}" run -W exceptions=y --env LLGO_WASM_SCHEDULER_REPANIC_TRACEBACK=1 "${work_dir}/scheduler-wasi.wasm"

# Exercise the CLI-level failure boundary in CI, not only the runners in
# isolation. The other public run calls below cover successful EC32, EC64,
# WC32, and alias execution through the same path.
wasm_ci_run_case EC32/emscripten public-runner-exit 1 1 0 0 0 \
expect_llgo_runner_failure emscripten j32 node "${scheduler_fixture}"
wasm_ci_run_case EC32/emscripten public-runner-timeout 1 0 1 0 0 \
	expect_llgo_runner_timeout emscripten j32 node "${scheduler_fixture}"

# Timers share the Go-derived heap but use different host-wait backends.
wasm_ci_run_case EC32/emscripten timers 1 0 0 0 0 \
	run_emscripten emscripten emscripten-runner.mjs "${timer_fixture}" "wasm timers ok" "timers-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 timers 1 0 0 0 0 \
	run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${timer_fixture}" "wasm timers ok" "timers-memory64"
wasm_ci_run_case WC32/wasi timers 1 0 0 0 0 \
	run_wasi wasi "${timer_fixture}" "wasm timers ok" "timers-wasi"

# R2 enables the non-moving collector by default for each canonical
# single-worker hosted target. This fixture covers active and suspended G roots,
# closures/interfaces/aggregates, panic/recover unwinding, pure-Go loop
# safepoints, reclamation, aligned allocation, and memory growth.
wasm_ci_run_case EC32/emscripten gc 1 0 0 0 0 \
	run_emscripten emscripten emscripten-runner.mjs "${gc_fixture}" "wasm gc ok" "gc-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 gc 1 0 0 0 0 \
	run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${gc_fixture}" "wasm gc ok" "gc-memory64"
wasm_ci_run_case WC32/wasi gc 1 0 0 0 0 \
	run_wasi wasi "${gc_fixture}" "wasm gc ok" "gc-wasi"

# Finalizers, cleanups, and weak references share the collector lifecycle but
# have additional ordering, cancellation, and dynamic-call ABI requirements.
wasm_ci_run_case EC32/emscripten lifecycle 1 0 0 0 0 \
	run_llgo_run emscripten "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 lifecycle 1 0 0 0 0 \
	run_llgo_run emscripten-memory64 "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-memory64"
wasm_ci_run_case WC32/wasi lifecycle 1 0 0 0 0 \
	run_llgo_run wasi "${lifecycle_fixture}" "wasm lifecycle ok" "lifecycle-wasi"

# A registered JS callback is a host wake source even when no Go timer exists.
# This catches treating an empty timer heap as an immediate deadlock.
wasm_ci_run_case EC32/emscripten callback 1 0 0 0 0 \
	run_emscripten emscripten emscripten-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 callback 1 0 0 0 0 \
	run_emscripten emscripten-memory64 emscripten-memory64-runner.mjs "${callback_fixture}" "wasm callback-only wake ok" "callback-memory64"

# A real browser must run both the named Emscripten provider and the raw J32
# GoJS provider. Reuse the named callback artifact and compile only one extra
# module so this gate does not duplicate the full Node matrix.
expect_browser_load_failure
run_browser "${work_dir}/callback-emscripten.mjs" "wasm callback-only wake ok"
env GOOS=js GOARCH=wasm "${llgo_cmd}" build -o "${work_dir}/callback-gojs.mjs" "${callback_fixture}"
wasm-tools validate --features all "${work_dir}/callback-gojs.wasm"
run_browser "${work_dir}/callback-gojs.mjs" "wasm callback-only wake ok"

# Keep the legacy named aliases executable while raw js/wasm remains the
# browser/worker-only compatibility path defined by R0.
wasm_ci_run_case L32/wasm-alias scheduler 1 0 0 0 0 \
	run_llgo_run wasm "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasm"
wasm_ci_run_case LW32/wasip1-alias scheduler 1 0 0 0 0 \
	run_llgo_run wasip1 "${scheduler_fixture}" "wasm scheduler ok" "scheduler-legacy-wasip1"

# Reuse both callback modules: no extra compilations for the JS boundary cases.
run_host_call_boundaries "${work_dir}/callback-emscripten.mjs"
run_host_call_boundaries "${work_dir}/callback-memory64.mjs"

fi

if [[ "${suite}" != "test-command" ]]; then
run_wasi_empty_heap
fi

if [[ "${suite}" == "all" || "${suite}" == "test-command" ]]; then
# Exercise test-main generation, process exit, verbose output, and host runners
# through the public test command. The JS-specific callback case also verifies
# that host readiness interrupts a longer Go timer wait without re-entering an
# arbitrary parked G.
wasm_ci_run_case EC32/emscripten public-test 2 0 0 0 0 \
	run_llgo_test emscripten "test-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 public-test 2 0 0 0 0 \
	run_llgo_test emscripten-memory64 "test-memory64"
wasm_ci_run_case WC32/wasi public-test 2 0 0 0 0 \
	run_llgo_test wasi "test-wasi"
wasm_ci_run_case EC32/emscripten compile-only 0 0 0 0 0 \
	run_llgo_test_compile_only emscripten "test-compile-only-emscripten"
wasm_ci_run_case EC64/emscripten-memory64 compile-only 0 0 0 0 0 \
	run_llgo_test_compile_only emscripten-memory64 "test-compile-only-memory64"
wasm_ci_run_case WC32/wasi compile-only 0 0 0 0 0 \
	run_llgo_test_compile_only wasi "test-compile-only-wasi"

# Raw GOOS/GOARCH selection must remain executable through the public command.
wasm_ci_run_case J32/gojs public-test 1 0 0 0 0 \
	run_llgo_go_profile_test js "test-gojs"
wasm_ci_run_case W32/wasi public-test 1 0 0 0 0 \
	run_llgo_go_profile_test wasip1 "test-gowasi" '^TestRawWasm(Runner|ReflectionBridge)$' "${runner_test_fixture}"
wasm_ci_run_case J32/gojs public-run 1 0 0 0 0 \
	run_llgo_go_profile_run js "run-gojs"
wasm_ci_run_case W32/wasi public-run 1 0 0 0 0 \
	run_llgo_go_profile_run wasip1 "run-gowasi"
fi

echo "single-worker WebAssembly ${suite} checks passed"
