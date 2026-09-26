#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/wasm_ci_report.sh"

test_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-ci-report.XXXXXX")"
trap 'rm -rf "${test_dir}"' EXIT
report_file="${test_dir}/records.tsv"
wasm_ci_report_init "${report_file}"

wasm_ci_run_case emscripten success 2 0 0 0 0 true
wasm_ci_run_case emscripten expected-exit 1 1 0 0 0 true
wasm_ci_run_case wasi timeout 1 0 1 0 0 true
wasm_ci_run_case wasi skipped 0 0 0 1 2 true

fail_mid_case() {
	false
	printf 'continued after failure\n' >"${test_dir}/continued"
}

set +e
wasm_ci_run_case emscripten broken 1 0 0 0 0 fail_mid_case
failure_status=$?
set -e
if [[ ${failure_status} -eq 0 ]]; then
	echo "failed report case returned success" >&2
	exit 1
fi
if [[ -e "${test_dir}/continued" ]]; then
	echo "report case continued after a failed command" >&2
	exit 1
fi

summary="$(wasm_ci_render_report "${report_file}" 1)"
assert_contains() {
	local expected="$1"
	local actual="$2"
	if ! grep -Fq "${expected}" <<<"${actual}"; then
		printf 'missing report entry %q in:\n%s\n' "${expected}" "${actual}" >&2
		exit 1
	fi
}
for expected in \
	'| emscripten | 3 | 3 | 1 | 0 | 1 | 0 | 0 |' \
	'| wasi | 2 | 1 | 0 | 1 | 0 | 1 | 2 |' \
	'| **Total** | **5** | **4** | **1** | **1** | **1** | **1** | **2** |' \
	'Suite result: fail (exit status 1).' \
	"Unexpected failure: \`emscripten/broken\`"; do
	assert_contains "${expected}" "${summary}"
done

github_summary="${test_dir}/github-summary.md"
# The test-command suite starts with a recorded case. Runtime-only preflight
# checks run before its first recorded case and would obscure this assertion.
set +e
integration_output="$(GITHUB_STEP_SUMMARY="${github_summary}" LLGO="${test_dir}/missing-llgo" \
	"${repo_root}/dev/test_wasm_single_worker.sh" test-command 2>&1)"
integration_status=$?
set -e
if [[ ! -f "${github_summary}" ]]; then
	printf 'single-worker report missing (exit %s):\n%s\n' "${integration_status}" "${integration_output}" >&2
	exit 1
fi
if [[ ${integration_status} -eq 0 ]]; then
	echo "single-worker suite with a missing compiler returned success" >&2
	exit 1
fi
github_output="$(cat "${github_summary}")"
for expected in \
	'| EC32/emscripten | 1 | 0 | 0 | 0 | 1 | 0 | 0 |' \
	'Suite result: fail' \
	"Unexpected failure: \`EC32/emscripten/public-test\`"; do
	assert_contains "${expected}" "${integration_output}"
	assert_contains "${expected}" "${github_output}"
done

echo "WebAssembly CI report checks passed"
