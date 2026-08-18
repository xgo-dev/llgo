#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/go_toolchain.sh"

if [[ $# -lt 2 ]]; then
	echo "usage: $0 <1.20|...|1.26|exact-version> <command> [argument ...]" >&2
	exit 2
fi
requested=$1
shift
if ! target_version="$(llgo_resolve_go_version "${repo_root}" "${requested}")"; then
	echo "unsupported Go version: ${requested}" >&2
	exit 2
fi
target_root="$(llgo_go_root "${target_version}")"

export PATH="${target_root}/bin:${PATH}"
export GOTOOLCHAIN=local
export GOENV=off
export GOFLAGS=

echo "Running with go${target_version}: $*"
"$@"
