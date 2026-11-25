#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/go_toolchain.sh"
cd "${repo_root}"

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <1.20|...|1.27|exact-version>" >&2
	exit 2
fi
if ! target_version="$(llgo_resolve_go_version "${repo_root}" "$1")"; then
	echo "unsupported Go version: $1" >&2
	exit 2
fi
target_minor="${target_version%.*}"
case "${target_minor}" in
	1.20|1.21|1.22|1.23)
		echo "wasm runtime compatibility starts at Go 1.24 (requires structs.HostLayout)" >&2
		exit 2
		;;
esac
target_root="$(llgo_go_root "${target_version}")"
target_go="$(llgo_go_binary "${target_root}")"

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-wasm-runtime.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT

llgo_cmd="${LLGO:-}"
if [[ -z "${llgo_cmd}" ]]; then
	dev/build_ci_tools.sh "${work_dir}/bin"
	llgo_cmd="${work_dir}/bin/llgo"
elif [[ "${llgo_cmd}" != */* ]]; then
	llgo_cmd="$(command -v "${llgo_cmd}")"
elif [[ "${llgo_cmd}" != /* ]]; then
	llgo_cmd="$(cd "$(dirname "${llgo_cmd}")" && pwd)/$(basename "${llgo_cmd}")"
fi

modfile="${work_dir}/wasm-runtime.mod"
cp .github/test-go.mod "${modfile}"
cp .github/test-go.sum "${work_dir}/wasm-runtime.sum"
GOTOOLCHAIN=local "${target_go}" mod edit -modfile="${modfile}" \
	-module="github.com/xgo-dev/llgo/cmd" \
	-go="${target_minor}" \
	-replace="github.com/xgo-dev/llgo/runtime=${repo_root}/runtime"

export PATH="${target_root}/bin:${PATH}"
export GOTOOLCHAIN=local
export GOWORK=off
export GOENV=off
export GOFLAGS=
export LLGO_ROOT="${repo_root}"
export LLGO_BUILD_CACHE="${LLGO_BUILD_CACHE:-off}"

echo "Building wasm runtime with go${target_version}"
(
	cd "${repo_root}/cmd"
	GOOS=js GOARCH=wasm "${llgo_cmd}" build -modfile="${modfile}" \
		-o "${work_dir}/runtime-js.wasm" ./compiler/build/testdata/wasm-runtime
	GOOS=wasip1 GOARCH=wasm "${llgo_cmd}" build -modfile="${modfile}" \
		-o "${work_dir}/runtime-wasip1.wasm" ./compiler/build/testdata/wasm-runtime
)
file "${work_dir}/runtime-js.wasm" "${work_dir}/runtime-wasip1.wasm"
