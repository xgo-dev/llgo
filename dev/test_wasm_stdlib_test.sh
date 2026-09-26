#!/usr/bin/env bash

set -euo pipefail

# Re-enter this file as a deterministic compiler stand-in. These checks test
# acceptance accounting, not the standard library itself.
if [[ "${1:-}" == test ]]; then
	mode="${LLGO_WASM_STDLIB_FAKE_MODE}"
	case "${mode}" in
		compiler-failure) exit 19 ;;
		empty-run) exit 0 ;;
	esac
	printf '%s\n' '--- PASS: TestBytesBuffer (0.00s)' PASS
	if [[ "${mode}" != missing-witness ]]; then
		printf '%s\n' '--- PASS: TestStringsBuilder (0.00s)'
	fi
	printf '%s\n' PASS
	if [[ "${mode}" != partial-run ]]; then
		printf '%s\n' '--- PASS: TestMarshalUnmarshal (0.00s)' PASS
	fi
	case "${mode}" in
		failed-test) printf '%s\n' '--- FAIL: TestUnexpected (0.00s)' ;;
		skipped-test) printf '%s\n' '--- SKIP: TestUnexpected (0.00s)' ;;
		leaked-artifact) touch "${TMPDIR}/leaked.wasm" ;;
	esac
	exit 0
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-stdlib-tests.XXXXXX")"
trap 'rm -rf "${test_dir}"' EXIT

for mode in pass compiler-failure empty-run partial-run missing-witness failed-test skipped-test leaked-artifact; do
	output="${test_dir}/${mode}.out"
	summary="${test_dir}/${mode}.md"
	status=0
	LLGO_WASM_STDLIB_FAKE_MODE="${mode}" \
		LLGO="${repo_root}/dev/test_wasm_stdlib_test.sh" \
		GITHUB_STEP_SUMMARY="${summary}" \
		bash "${repo_root}/dev/test_wasm_stdlib.sh" >"${output}" 2>&1 || status=$?
	if [[ "${mode}" == pass ]]; then
		if [[ ${status} -ne 0 ]]; then
			cat "${output}"
			echo "complete standard-library run failed" >&2
			exit 1
		fi
		expected='| **Total** | **3** | **9** | **0** | **0** | **0** | **0** | **0** |'
	else
		if [[ ${status} -eq 0 ]]; then
			echo "standard-library acceptance incorrectly passed: ${mode}" >&2
			exit 1
		fi
		expected='| **Total** | **1** | **0** | **0** | **0** | **1** | **0** | **0** |'
		# shellcheck disable=SC2016 # Match literal Markdown backticks.
		grep -Fq 'Unexpected failure: `EC32/emscripten/stdlib-bytes-strings-json`' "${summary}"
	fi
	grep -Fq "${expected}" "${output}"
	grep -Fq "${expected}" "${summary}"
done

echo "WebAssembly standard-library acceptance checks passed"
