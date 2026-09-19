#!/usr/bin/env bash

set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${root_dir}/dev/go_toolchain.sh"
cd "${root_dir}"

usage() {
	echo "usage: $0 <1.20|...|1.27|exact-version> [package ...]" >&2
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

tool_suffix=
if [[ "${OS:-}" == "Windows_NT" ]]; then
	tool_suffix=.exe
fi

llgo_cmd="${LLGO:-}"
if [[ -z "${llgo_cmd}" ]]; then
	build_ci_tools
	llgo_cmd="${work_dir}/bin/llgo${tool_suffix}"
elif [[ "${llgo_cmd}" != */* ]]; then
	llgo_cmd="$(command -v "${llgo_cmd}")"
elif [[ "${llgo_cmd}" != /* ]]; then
	llgo_cmd="$(cd "$(dirname "${llgo_cmd}")" && pwd)/$(basename "${llgo_cmd}")"
fi

check_std_symbols="${CHECK_STD_SYMBOLS:-}"
if [[ "${LLGO_TEST_CHECK_SYMBOLS:-}" == 1 && -z "${check_std_symbols}" ]]; then
	build_ci_tools
	check_std_symbols="${work_dir}/bin/check_std_symbols${tool_suffix}"
fi

target_root="$(llgo_go_root "${target_version}")"
target_go="$(llgo_go_binary "${target_root}")"
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
test_llgo_cmd="${llgo_cmd}"
if [[ "${OS:-}" == "Windows_NT" ]] && command -v cygpath >/dev/null 2>&1; then
	test_llgo_cmd="$(cygpath -w "${llgo_cmd}")"
fi
export LLGO_TEST_LLGO="${test_llgo_cmd}"
export LLGO_TEST_COMPILER="${test_llgo_cmd}"
export LLGO_TEST_MODFILE="${modfile}"
# LLGo's shared cache may contain standard-library objects from another Go
# release. CI jobs are isolated and may opt back in explicitly.
export LLGO_BUILD_CACHE="${LLGO_BUILD_CACHE:-off}"

requested_packages=("$@")
if [[ "${#requested_packages[@]}" -eq 0 ]]; then
	case "${LLGO_TEST_SUITE:-}" in
		core)
			requested_packages=(./test/...)
			;;
		std)
			case "${target_minor}" in
				1.20|1.21)
					requested_packages=(./test/std/bufio ./test/std/bytes ./test/std/encoding/json ./test/std/math/bits)
					;;
				1.22)
					requested_packages=(./test/std/bufio ./test/std/bytes ./test/std/encoding/json ./test/std/go/version)
					;;
				1.23)
					requested_packages=(./test/std/iter ./test/std/maps ./test/std/slices ./test/std/structs ./test/std/unique)
					;;
				1.24)
					requested_packages=(./test/std/bytes ./test/std/crypto/hkdf ./test/std/crypto/pbkdf2 ./test/std/weak)
					;;
				1.25)
					requested_packages=(
						./test/std/crypto
						./test/std/crypto/ecdsa
						./test/std/crypto/sha3
						./test/std/go/ast
						./test/std/go/token
						./test/std/go/types
						./test/std/hash
						./test/std/hash/maphash
						./test/std/io/fs
						./test/std/log/slog
						./test/std/mime/multipart
						./test/std/net/http
						./test/std/os
						./test/std/reflect
						./test/std/runtime/trace
						./test/std/sync
						./test/std/testing
						./test/std/testing/fstest
						./test/std/testing/synctest
						./test/std/unicode
					)
					;;
				1.26)
					requested_packages=(
						./test/std/bytes
						./test/std/crypto
						./test/std/crypto/ecdh
						./test/std/crypto/fips140
						./test/std/crypto/hpke
						./test/std/crypto/mlkem
						./test/std/crypto/mlkem/mlkemtest
						./test/std/crypto/rsa
						./test/std/crypto/x509
						./test/std/errors
						./test/std/go/ast
						./test/std/go/token
						./test/std/log/slog
						./test/std/net
						./test/std/net/http
						./test/std/net/netip
						./test/std/os
						./test/std/reflect
						./test/std/testing
						./test/std/testing/cryptotest
					)
					;;
				*) requested_packages=(./test/std/...) ;;
			esac
			;;
		*)
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
				1.25)
					# Cover every package with Go 1.25-specific symbol checks without
					# repeating the full test tree reserved for the primary release.
					requested_packages=(
						./test/std/crypto
						./test/std/crypto/ecdsa
						./test/std/crypto/sha3
						./test/std/go/ast
						./test/std/go/token
						./test/std/go/types
						./test/std/hash
						./test/std/hash/maphash
						./test/std/io/fs
						./test/std/log/slog
						./test/std/mime/multipart
						./test/std/net/http
						./test/std/os
						./test/std/reflect
						./test/std/runtime/trace
						./test/std/sync
						./test/std/testing
						./test/std/testing/fstest
						./test/std/testing/synctest
						./test/std/unicode
						./test/goroot
					)
					;;
				1.26)
					# Keep the compatibility lane focused on packages that contain Go
					# 1.26-specific checks. Go 1.27 owns the complete test matrix.
					requested_packages=(
						./test/std/bytes
						./test/std/crypto
						./test/std/crypto/ecdh
						./test/std/crypto/fips140
						./test/std/crypto/hpke
						./test/std/crypto/mlkem
						./test/std/crypto/mlkem/mlkemtest
						./test/std/crypto/rsa
						./test/std/crypto/x509
						./test/std/errors
						./test/std/go/ast
						./test/std/go/token
						./test/std/log/slog
						./test/std/net
						./test/std/net/http
						./test/std/net/netip
						./test/std/os
						./test/std/reflect
						./test/std/testing
						./test/std/testing/cryptotest
						./test/goroot
					)
					;;
				*) requested_packages=(./test/...) ;;
			esac
			;;
	esac
fi

packages_file="${work_dir}/packages.txt"
go list -modfile="${modfile}" -tags=llgo "${requested_packages[@]}" | sort -u >"${packages_file}"
packages=()
while IFS= read -r package; do
	if [[ "${LLGO_TEST_SUITE:-}" == "core" ]]; then
		case "${package}" in
			github.com/xgo-dev/llgo/test/goroot|github.com/xgo-dev/llgo/test/goroot/*)
				continue
				;;
			github.com/xgo-dev/llgo/test/std/bufio|\
			github.com/xgo-dev/llgo/test/std/bytes|\
			github.com/xgo-dev/llgo/test/std/encoding/binary|\
			github.com/xgo-dev/llgo/test/std/encoding/json|\
			github.com/xgo-dev/llgo/test/std/errors|\
			github.com/xgo-dev/llgo/test/std/fmt|\
			github.com/xgo-dev/llgo/test/std/io|\
			github.com/xgo-dev/llgo/test/std/math/bits|\
			github.com/xgo-dev/llgo/test/std/sort|\
			github.com/xgo-dev/llgo/test/std/strconv|\
			github.com/xgo-dev/llgo/test/std/strings|\
			github.com/xgo-dev/llgo/test/std/sync|\
			github.com/xgo-dev/llgo/test/std/sync/*)
				;;
			github.com/xgo-dev/llgo/test/std|github.com/xgo-dev/llgo/test/std/*)
				continue
				;;
		esac
	elif [[ "${LLGO_TEST_SUITE:-}" == "std" ]]; then
		case "${package}" in
			github.com/xgo-dev/llgo/test/std|github.com/xgo-dev/llgo/test/std/*)
				;;
			*)
				continue
				;;
		esac
	fi
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

test_flags=(-timeout="${LLGO_TEST_TIMEOUT:-20m}" -modfile="${modfile}")
if [[ -n "${LLGO_TEST_JOBS:-}" ]]; then
	test_flags=(-p="${LLGO_TEST_JOBS}" "${test_flags[@]}")
fi
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
