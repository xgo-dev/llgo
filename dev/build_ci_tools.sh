#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/go_toolchain.sh"

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <output-directory>" >&2
	exit 2
fi

output_dir=$1
mkdir -p "${output_dir}"
output_dir="$(cd "${output_dir}" && pwd)"

build_version="$(llgo_resolve_go_version "${repo_root}" 1.27)"
build_root="$(llgo_go_root "${build_version}")"
build_go="$(llgo_go_binary "${build_root}")"

tool_suffix=
if [[ "${OS:-}" == "Windows_NT" ]]; then
	tool_suffix=.exe
fi

echo "Building LLGo tools with go${build_version} into ${output_dir}"
(
	cd "${repo_root}"
	GOBIN="${output_dir}" GOTOOLCHAIN=local "${build_go}" install ./cmd/...
	if [[ "${LLGO_BUILD_GOROOT_RUNNER:-}" == 1 ]]; then
		GOTOOLCHAIN=local "${build_go}" build -tags=dev \
			-o "${output_dir}/llgo${tool_suffix}" ./cmd/llgo
		GOTOOLCHAIN=local "${build_go}" test -c \
			-o "${output_dir}/goroot-runner${tool_suffix}" ./test/goroot
	fi
)
