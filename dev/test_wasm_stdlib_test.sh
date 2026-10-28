#!/usr/bin/env bash

set -euo pipefail

# Re-enter this file as a deterministic compiler stand-in. These checks test
# acceptance accounting, not the standard library itself.
if [[ "${1:-}" == test ]]; then
	mode="${LLGO_WASM_STDLIB_FAKE_MODE}"
	args=("$@")
	target=""
	for ((index=0; index<${#args[@]}; index++)); do
		if [[ "${args[index]}" == -target ]]; then
			target="${args[index+1]}"
		fi
	done
	printf '%s\n' "${target}" >>"${LLGO_WASM_STDLIB_CALLS}"
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

for selection in all wasi emscripten emscripten-memory64; do
	args=()
	expected_targets=$'emscripten\nemscripten-memory64\nwasi'
	checks=3
	first=emscripten
	if [[ "${selection}" != all ]]; then
		args=("${selection}")
		expected_targets="${selection}"
		checks=1
		first="${selection}"
	fi
	case "${first}" in
	wasi) profile=WC32/wasi ;;
	emscripten) profile=EC32/emscripten ;;
	emscripten-memory64) profile=EC64/emscripten-memory64 ;;
	esac
	for mode in pass compiler-failure empty-run partial-run missing-witness failed-test skipped-test leaked-artifact; do
		output="${test_dir}/${selection}-${mode}.out"
		summary="${test_dir}/${selection}-${mode}.md"
		calls="${test_dir}/${selection}-${mode}.calls"
		: >"${calls}"
		status=0
		LLGO_WASM_STDLIB_FAKE_MODE="${mode}" \
			LLGO_WASM_STDLIB_CALLS="${calls}" \
			LLGO="${repo_root}/dev/test_wasm_stdlib_test.sh" \
			GITHUB_STEP_SUMMARY="${summary}" \
			bash "${repo_root}/dev/test_wasm_stdlib.sh" "${args[@]}" >"${output}" 2>&1 || status=$?
		if [[ "${mode}" == pass ]]; then
			if [[ ${status} -ne 0 ]]; then
				cat "${output}"
				echo "complete standard-library run failed" >&2
				exit 1
			fi
			expected="| **Total** | **${checks}** | **$((checks*3))** | **0** | **0** | **0** | **0** | **0** |"
			if [[ "$(cat "${calls}")" != "${expected_targets}" ]]; then
				echo "standard-library target selection duplicated or omitted targets: ${selection}" >&2
				cat "${calls}"
				exit 1
			fi
		else
			if [[ ${status} -eq 0 ]]; then
				echo "standard-library acceptance incorrectly passed: ${mode}" >&2
				exit 1
			fi
			expected='| **Total** | **1** | **0** | **0** | **0** | **1** | **0** | **0** |'
			# shellcheck disable=SC2016 # Match literal Markdown backticks.
			grep -Fq "Unexpected failure: \`${profile}/stdlib-bytes-strings-json\`" "${summary}"
			[[ "$(cat "${calls}")" == "${first}" ]]
		fi
		grep -Fq "${expected}" "${output}"
		grep -Fq "${expected}" "${summary}"
	done
done

for invalid in unknown empty multiple; do
	case "${invalid}" in
	unknown) args=(wasm) ;;
	empty) args=("") ;;
	multiple) args=(wasi emscripten) ;;
	esac
	calls="${test_dir}/invalid-${invalid}.calls"
	summary="${test_dir}/invalid-${invalid}.md"
	: >"${calls}"
	status=0
	LLGO_WASM_STDLIB_FAKE_MODE=pass LLGO_WASM_STDLIB_CALLS="${calls}" \
		LLGO="${repo_root}/dev/test_wasm_stdlib_test.sh" GITHUB_STEP_SUMMARY="${summary}" \
		bash "${repo_root}/dev/test_wasm_stdlib.sh" "${args[@]}" >"${test_dir}/invalid-${invalid}.out" 2>&1 || status=$?
	if [[ ${status} -ne 2 || -s "${calls}" || -e "${summary}" ]]; then
		echo "invalid target selection reached compilation/accounting: ${invalid} (${status})" >&2
		exit 1
	fi
done

echo "WebAssembly standard-library acceptance checks passed"
