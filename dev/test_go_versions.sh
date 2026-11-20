#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${repo_root}/dev/go_toolchain.sh"
cd "${repo_root}"

versions=("$@")
if [[ ${#versions[@]} -eq 0 ]]; then
	read -r -a versions <<< "$(llgo_supported_go_releases)"
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

failed_versions=()
for version in "${versions[@]}"; do
	echo
	echo "==== test/ with Go ${version} ===="
	# LLGo's cache contains target-standard-library objects. When callers opt in
	# with LLGO_BUILD_CACHE=1, keep reuse from crossing Go versions.
	version_cache_dir="${work_dir}/cache/go${version}"
	mkdir -p "${version_cache_dir}"
	std_buildmodes="${LLGO_TEST_STD_BUILDMODES:-}"
	if [[ -z "${std_buildmodes}" ]]; then
		case "${version}" in
			1.27|1.27.*) std_buildmodes=1 ;;
			*) std_buildmodes=0 ;;
		esac
	fi
	if ! XDG_CACHE_HOME="${version_cache_dir}" \
		LLGO="${llgo_cmd}" \
		CHECK_STD_SYMBOLS="${check_std_symbols}" \
		LLGO_TEST_BENCH_GO126="${LLGO_TEST_BENCH_GO126:-1}" \
		LLGO_TEST_CHECK_SYMBOLS="${check_symbols}" \
		LLGO_TEST_STD_BUILDMODES="${std_buildmodes}" \
		dev/test_go_version.sh "${version}"; then
		failed_versions+=("${version}")
	fi
done

if [[ ${#failed_versions[@]} -ne 0 ]]; then
	printf 'versioned test failures:' >&2
	printf ' Go %s' "${failed_versions[@]}" >&2
	printf '\n' >&2
	exit 1
fi
