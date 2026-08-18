#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

versions=("$@")
if [[ ${#versions[@]} -eq 0 ]]; then
	versions=(1.20 1.21 1.22 1.23 1.24 1.25 1.26)
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-test-go-versions.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT

llgo_cmd="${LLGO:-}"
check_std_symbols="${CHECK_STD_SYMBOLS:-}"
check_symbols="${LLGO_TEST_CHECK_SYMBOLS:-1}"
if [[ -z "${llgo_cmd}" || ( "${check_symbols}" == 1 && -z "${check_std_symbols}" ) ]]; then
	dev/build_ci_tools.sh "${work_dir}/bin"
	if [[ -z "${llgo_cmd}" ]]; then
		llgo_cmd="${work_dir}/bin/llgo"
	fi
	if [[ -z "${check_std_symbols}" ]]; then
		check_std_symbols="${work_dir}/bin/check_std_symbols"
	fi
fi

for version in "${versions[@]}"; do
	echo
	echo "==== test/ with Go ${version} ===="
	std_buildmodes="${LLGO_TEST_STD_BUILDMODES:-}"
	if [[ -z "${std_buildmodes}" ]]; then
		case "${version}" in
			1.26|1.26.*) std_buildmodes=1 ;;
			*) std_buildmodes=0 ;;
		esac
	fi
	LLGO="${llgo_cmd}" \
		CHECK_STD_SYMBOLS="${check_std_symbols}" \
		LLGO_TEST_BENCH_GO126="${LLGO_TEST_BENCH_GO126:-1}" \
		LLGO_TEST_CHECK_SYMBOLS="${check_symbols}" \
		LLGO_TEST_STD_BUILDMODES="${std_buildmodes}" \
		dev/test_go_version.sh "${version}"
done
