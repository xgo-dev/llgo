#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${root_dir}/dev/go_toolchain.sh"
cd "${root_dir}"

usage() {
	echo "usage: $0 <1.20|...|1.26|exact-version> [package ...]" >&2
	exit 2
}

if [[ $# -eq 0 ]]; then
	usage
fi

requested="$1"
shift
if ! target_version="$(llgo_resolve_go_version "${root_dir}" "${requested}")"; then
	usage
fi
target_minor="${target_version%.*}"

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/llgo-test-go.XXXXXX")"
trap 'rm -rf "${work_dir}"' EXIT

tools_built=0
build_ci_tools() {
	if [[ "${tools_built}" == 1 ]]; then
		return
	fi
	dev/build_ci_tools.sh "${work_dir}/bin"
	tools_built=1
}

llgo_cmd="${LLGO:-}"
if [[ -z "${llgo_cmd}" ]]; then
	build_ci_tools
	llgo_cmd="${work_dir}/bin/llgo"
elif [[ "${llgo_cmd}" != */* ]]; then
	llgo_cmd="$(command -v "${llgo_cmd}")"
elif [[ "${llgo_cmd}" != /* ]]; then
	llgo_cmd="$(cd "$(dirname "${llgo_cmd}")" && pwd)/$(basename "${llgo_cmd}")"
fi

llgen_cmd="${LLGO_TEST_LLGEN:-}"
if [[ -z "${llgen_cmd}" && -x "$(dirname "${llgo_cmd}")/llgen" ]]; then
	llgen_cmd="$(dirname "${llgo_cmd}")/llgen"
fi
if [[ -z "${llgen_cmd}" ]]; then
	build_ci_tools
	llgen_cmd="${work_dir}/bin/llgen"
fi

check_std_symbols="${CHECK_STD_SYMBOLS:-}"
if [[ "${LLGO_TEST_CHECK_SYMBOLS:-}" == 1 && -z "${check_std_symbols}" ]]; then
	build_ci_tools
	check_std_symbols="${work_dir}/bin/check_std_symbols"
fi

target_root="$(llgo_go_root "${target_version}")"
target_go="${target_root}/bin/go"
actual_version="$(cd "${work_dir}" && GOTOOLCHAIN=local "${target_go}" env GOVERSION)"
if [[ "${actual_version}" != "go${target_version}" ]]; then
	echo "expected go${target_version}, got ${actual_version}" >&2
	exit 1
fi

modfile="${work_dir}/test.mod"
cp .github/test-go.mod "${modfile}"
cp .github/test-go.sum "${work_dir}/test.sum"
GOTOOLCHAIN=local "${target_go}" mod edit \
	-modfile="${modfile}" \
	-go="${target_minor}" \
	-replace="github.com/xgo-dev/llgo/runtime=${root_dir}/runtime"

export PATH="${target_root}/bin:${PATH}"
export GOTOOLCHAIN=local
export GOWORK=off
export GOENV=off
export GOFLAGS=
export LLGO_ROOT="${root_dir}"
export LLGO_TEST_LLGO="${llgo_cmd}"
export LLGO_TEST_LLGEN="${llgen_cmd}"
export LLGO_TEST_MODFILE="${modfile}"
# LLGo's shared cache may contain standard-library objects from another Go
# release. CI jobs are isolated and may opt back in explicitly.
export LLGO_BUILD_CACHE="${LLGO_BUILD_CACHE:-off}"

requested_packages=("$@")
if [[ "${#requested_packages[@]}" -eq 0 ]]; then
	case "${target_minor}" in
		1.20|1.21)
			requested_packages=(./test/std/bufio ./test/std/bytes ./test/std/encoding/json ./test/std/math/bits ./test/goroot)
			;;
		1.22)
			requested_packages=(./test/std/bufio ./test/std/bytes ./test/std/encoding/json ./test/std/go/version ./test/goroot)
			;;
		1.23)
			requested_packages=(./test/std/iter ./test/std/maps ./test/std/slices ./test/std/structs ./test/std/unique ./test/goroot)
			;;
		1.24)
			requested_packages=(./test/std/bytes ./test/std/crypto/hkdf ./test/std/crypto/pbkdf2 ./test/std/weak ./test/goroot)
			;;
		*) requested_packages=(./test/...) ;;
	esac
fi

packages_file="${work_dir}/packages.txt"
go list -modfile="${modfile}" -tags=llgo "${requested_packages[@]}" | sort -u >"${packages_file}"
packages=()
while IFS= read -r package; do
	packages+=("${package}")
done <"${packages_file}"

shard_index="${SHARD_INDEX:-0}"
shard_total="${SHARD_TOTAL:-1}"
if (( shard_total < 1 || shard_index < 0 || shard_index >= shard_total )); then
	echo "invalid shard ${shard_index}/${shard_total}" >&2
	exit 2
fi
selected=()
for i in "${!packages[@]}"; do
	if (( i % shard_total == shard_index )); then
		selected+=("${packages[$i]}")
	fi
done
if [[ "${#selected[@]}" -eq 0 ]]; then
	echo "no packages selected for shard ${shard_index}/${shard_total}" >&2
	exit 1
fi

echo "Go toolchain: ${actual_version} (${target_root})"
echo "LLGo: ${llgo_cmd}"
echo "Shard: ${shard_index}/${shard_total}; packages: ${#selected[@]}"
printf '  %s\n' "${selected[@]}"

test_flags=(-p="${LLGO_TEST_JOBS:-4}" -timeout="${LLGO_TEST_TIMEOUT:-20m}" -modfile="${modfile}")
if [[ "${LLGO_TEST_COMPILE_ONLY:-}" == 1 ]]; then
	test_flags+=(-run='^$')
fi
if [[ "${LLGO_TEST_BENCH_GO126:-}" == 1 && "${target_minor}" == 1.26 ]]; then
	test_flags+=(-bench='^BenchmarkGo126' -benchtime=1x)
fi
SECONDS=0
"${llgo_cmd}" test "${test_flags[@]}" "${selected[@]}"
echo "llgo test completed in ${SECONDS}s"

std_packages=()
for package in "${selected[@]}"; do
	case "${package}" in
		github.com/xgo-dev/llgo/test/std/*) std_packages+=("${package}") ;;
	esac
done

if [[ "${LLGO_TEST_CHECK_SYMBOLS:-}" == 1 && "${#std_packages[@]}" -ne 0 ]]; then
	SECONDS=0
	LLGO_TEST_MODFILE="${modfile}" \
		CHECK_STD_SYMBOLS="${check_std_symbols}" \
		doc/_readme/scripts/check_std_cover.sh "${std_packages[@]}"
	echo "standard-library symbol check completed in ${SECONDS}s"
fi
if [[ "${LLGO_TEST_STD_BUILDMODES:-}" == 1 && "${#std_packages[@]}" -ne 0 ]]; then
	SECONDS=0
	LLGO="${llgo_cmd}" LLGO_TEST_MODFILE="${modfile}" \
		dev/test_std_buildmodes.sh "${std_packages[@]}"
	echo "standard-library build-mode checks completed in ${SECONDS}s"
fi
