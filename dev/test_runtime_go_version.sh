#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <1.20|...|1.26|exact-version>" >&2
	exit 2
fi

(
	cd "${repo_root}/runtime"
	"${repo_root}/dev/with_go_version.sh" "$1" go test ./...
)
