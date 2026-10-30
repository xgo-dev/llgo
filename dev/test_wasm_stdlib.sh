#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
if [[ $# -gt 1 ]]; then
	echo "usage: test_wasm_stdlib.sh [wasi|emscripten|emscripten-memory64]" >&2
	exit 2
fi
targets=(emscripten emscripten-memory64 wasi)
if [[ $# -eq 1 ]]; then
	case "$1" in
	wasi|emscripten|emscripten-memory64) targets=("$1") ;;
	*)
		echo "unknown WebAssembly standard-library target: $1" >&2
		exit 2
		;;
	esac
fi
source "${repo_root}/dev/wasm_ci_report.sh"
llgo_cmd="${LLGO:-llgo}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-stdlib.XXXXXX")"
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

# Run complete repository test packages, including their version-selected
# tests. This is an initial acceptance set, not a stdlib completeness claim.
packages=(
	"${repo_root}/test/std/bytes"
	"${repo_root}/test/std/strings"
	"${repo_root}/test/std/encoding/json"
)

run_stdlib() {
	local target="$1"
	local output="${work_dir}/${target}.out"
	local temp_dir="${work_dir}/${target}-tmp"
	mkdir -p "${temp_dir}"
	echo "testing bytes, strings, and encoding/json on ${target}"
	env TMPDIR="${temp_dir}" "${llgo_cmd}" test -target "${target}" -emulator \
		-v -count=1 -timeout=60s "${packages[@]}" 2>&1 | tee "${output}"

	# Each independently executed test binary prints one terminal PASS. Check
	# witnesses from all three packages so an empty or partial run cannot pass.
	local passed
	passed="$(grep -c '^PASS$' "${output}" || true)"
	if [[ "${passed}" != "${#packages[@]}" ]]; then
		echo "expected ${#packages[@]} completed test packages, got ${passed}" >&2
		return 1
	fi
	local test_name
	for test_name in TestBytesBuffer TestStringsBuilder TestMarshalUnmarshal; do
		grep -Fq -- "--- PASS: ${test_name} (" "${output}"
	done
	if grep -Eq -- '^[[:space:]]*--- (FAIL|SKIP):' "${output}"; then
		echo "standard-library acceptance contains failed or skipped tests" >&2
		return 1
	fi
	local artifact
	artifact="$(find "${temp_dir}" -type f \( -name '*.mjs' -o -name '*.wasm' \) -print -quit)"
	if [[ -n "${artifact}" ]]; then
		echo "implicit WebAssembly artifact was not removed: ${artifact}" >&2
		return 1
	fi
}

for target in "${targets[@]}"; do
	case "${target}" in
	emscripten) profile=EC32/emscripten ;;
	emscripten-memory64) profile=EC64/emscripten-memory64 ;;
	wasi) profile=WC32/wasi ;;
	esac
	wasm_ci_run_case "${profile}" stdlib-bytes-strings-json 3 0 0 0 0 run_stdlib "${target}"
done
