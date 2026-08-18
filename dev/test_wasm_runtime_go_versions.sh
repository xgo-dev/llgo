#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

versions=("$@")
if [[ ${#versions[@]} -eq 0 ]]; then
	versions=(1.24 1.26)
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-runtime-versions.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT

llgo_cmd="${LLGO:-}"
if [[ -z "${llgo_cmd}" ]]; then
	dev/build_ci_tools.sh "${work_dir}/bin"
	llgo_cmd="${work_dir}/bin/llgo"
fi

for version in "${versions[@]}"; do
	echo
	echo "==== wasm runtime with Go ${version} ===="
	LLGO="${llgo_cmd}" dev/test_wasm_runtime_go_version.sh "${version}"
done
