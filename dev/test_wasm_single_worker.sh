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
runner_test_fixture="${repo_root}/internal/build/testdata/wasm-runner-test"
runner_run_fixture="${repo_root}/internal/build/testdata/wasm-runner-run"
suite="${1:-all}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-single-worker.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
export LLGO_WASM_TEST_ENV=wasm-env-ok

case "${suite}" in
all | runtime | test-command) ;;
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

run_browser() {
	local module="$1"
	local expected="$2"
	run_with_timeout_limit 90s "${node_cmd}" "${repo_root}/dev/test_wasm_browser.mjs" \
		"${module}" "${expected}"
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
	local test_args=(-v -count=1 -timeout=30s)
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

if [[ "${suite}" != "test-command" ]]; then
# Canonical hosted targets exercise the same scheduler semantics under J32
# Emscripten, J64 Emscripten Memory64, and W32 WASI Preview 1.
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
# single-worker hosted target. This fixture covers active and suspended G roots,
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

# A real browser must run both the named Emscripten provider and the raw J32
# GoJS provider. Reuse the named callback artifact and compile only one extra
# module so this gate does not duplicate the full Node matrix.
run_browser "${work_dir}/callback-emscripten.mjs" "wasm callback-only wake ok"
env GOOS=js GOARCH=wasm "${llgo_cmd}" build -o "${work_dir}/callback-gojs.mjs" "${callback_fixture}"
wasm-tools validate --features all "${work_dir}/callback-gojs.wasm"
run_browser "${work_dir}/callback-gojs.mjs" "wasm callback-only wake ok"

# Reuse both callback modules: no extra compilations for the JS boundary cases.
run_host_call_boundaries "${work_dir}/callback-emscripten.mjs"
run_host_call_boundaries "${work_dir}/callback-memory64.mjs"

fi

if [[ "${suite}" != "runtime" ]]; then
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
# Raw GOOS/GOARCH selection must remain executable through the public command,
# not merely compile under the named C-ABI targets. GoJS runs the complete JS
# callback and filesystem set; Go WASI uses the smaller host-neutral subset to
# cover its automatic runner without duplicating the named-WASI acceptance.
run_llgo_go_profile_test js "test-gojs"
run_llgo_go_profile_test wasip1 "test-gowasi" '^TestRawWasmRunner$' "${runner_test_fixture}"
run_llgo_go_profile_run js "run-gojs"
run_llgo_go_profile_run wasip1 "run-gowasi"
fi

echo "single-worker WebAssembly ${suite} checks passed"
