#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

versions=("$@")
if [[ ${#versions[@]} -eq 0 ]]; then
	versions=(1.20 1.26)
fi

for version in "${versions[@]}"; do
	echo
	echo "==== runtime module with Go ${version} ===="
	dev/test_runtime_go_version.sh "${version}"
done
