#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
llgo_cmd="${LLGO:-llgo}"
node_cmd="${NODE:-node}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-stack.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT
if [[ $# -eq 0 ]]; then
	set -- emscripten emscripten-memory64
fi
for target in "$@"; do
	case "${target}" in
		emscripten|emscripten-memory64) ;;
		*) echo "unsupported stack-check target: ${target}" >&2; exit 2 ;;
	esac
	runner="${repo_root}/targets/${target}-runner.mjs"
	for test_case in 128KB:overflow 1MB:success; do
		size="${test_case%%:*}"
		expected="${test_case#*:}"
		module="${work_dir}/${target}-${size}.mjs"
		"${llgo_cmd}" build -target "${target}" -goroutine-stack-size="${size}" \
			-o "${module}" "${repo_root}/internal/build/testdata/wasm-stack-bounds"
		status=0
		if command -v timeout >/dev/null 2>&1; then
			timeout 60s "${node_cmd}" "${runner}" "${module}" > "${module}.log" 2>&1 || status=$?
		else
			"${node_cmd}" "${runner}" "${module}" > "${module}.log" 2>&1 || status=$?
		fi
		if [[ "${expected}" == overflow ]]; then
			if [[ ${status} -eq 0 ]] || ! grep -Fq 'Aborted(stack overflow' "${module}.log"; then
				cat "${module}.log"
				echo "${target}: expected checked stack overflow, got ${status}" >&2
				exit 1
			fi
		else
			if [[ ${status} -ne 0 ]] || ! grep -Fq 'wasm stack bounds ok' "${module}.log"; then
				cat "${module}.log"
				echo "${target}: larger goroutine stack failed with ${status}" >&2
				exit 1
			fi
		fi
		echo "${target}: ${size} stack check passed"
	done
done
