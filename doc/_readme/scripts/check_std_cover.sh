#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${ROOT_DIR}"

go_list() {
  if [[ -n "${LLGO_TEST_MODFILE:-}" ]]; then
    command go list "-modfile=${LLGO_TEST_MODFILE}" -tags=llgo "$@"
  else
    command go list -tags=llgo "$@"
  fi
}

module_path="$(go_list -m)"

packages=()
if [[ $# -eq 0 ]]; then
  package_output="$(go_list ./test/std/... | sort)"
  while IFS= read -r pkg; do
    [[ -n "${pkg}" ]] && packages+=("${pkg}")
  done <<< "${package_output}"
else
  package_output="$(go_list "$@" | sort -u)"
  while IFS= read -r pkg; do
    [[ -n "${pkg}" ]] && packages+=("${pkg}")
  done <<< "${package_output}"
fi

if [ "${#packages[@]}" -eq 0 ]; then
  echo "No stdlib test packages discovered under test/std" >&2
  exit 0
fi

args=()
covered_packages=()
for pkg in "${packages[@]}"; do
  rel_path="${pkg#"${module_path}"/}"
  if [[ "${rel_path}" != test/std/* ]]; then
    continue
  fi
  stdlib_pkg="${rel_path#test/std/}"
  covered_packages+=("${stdlib_pkg}")
  # abicall contains supplemental LLGo runtime/host ABI regressions, rather
  # than tests for a public standard-library package named runtime/abicall.
  if [[ "${stdlib_pkg}" == "runtime" || "${stdlib_pkg}" == "runtime/abicall" ]]; then
    continue
  fi
  # These Go 1.27 packages currently have focused compatibility tests. Keep
  # full upstream symbol coverage disabled until their complete suites land.
  if [[ "${stdlib_pkg}" == "crypto/mldsa" || "${stdlib_pkg}" == "encoding/json/jsontext" ||
        "${stdlib_pkg}" == "encoding/json/v2" || "${stdlib_pkg}" == "uuid" ]]; then
    continue
  fi
  args+=("-pkg" "${stdlib_pkg}")
done

if [[ $# -eq 0 ]]; then
  expected_file="$(mktemp)"
  covered_file="$(mktemp)"
  trap 'rm -f "${expected_file}" "${covered_file}"' EXIT

  # The std pattern may also report a package whose every source file is
  # excluded by the current platform. Require coverage only for packages that
  # have buildable Go or cgo sources under the selected toolchain/modfile.
  go_list -e \
    -f '{{if or .GoFiles .CgoFiles}}{{.ImportPath}}{{end}}' std \
    | awk 'NF' \
    | awk '!/(^|\/)internal(\/|$)/ && !/(^|\/)vendor(\/|$)/' \
    | sort -u > "${expected_file}"
  printf '%s\n' "${covered_packages[@]}" | sort -u > "${covered_file}"

  missing_packages="$(comm -23 "${expected_file}" "${covered_file}")"
  if [[ -n "${missing_packages}" ]]; then
    echo "Public standard-library packages missing test/std coverage:" >&2
    while IFS= read -r pkg; do
      echo "  - ${pkg}" >&2
    done <<< "${missing_packages}"
    exit 1
  fi

  expected_count="$(wc -l < "${expected_file}" | tr -d ' ')"
  covered_count="$(wc -l < "${covered_file}" | tr -d ' ')"
  echo "Public standard-library package coverage: ${covered_count}/${expected_count}"
fi

if [[ "${#args[@]}" -eq 0 ]]; then
  echo "No standard-library symbols selected for coverage checking"
  exit 0
fi

check_command=()
if [[ -n "${CHECK_STD_SYMBOLS:-}" ]]; then
  check_command+=("${CHECK_STD_SYMBOLS}")
else
  check_command+=(go run ./chore/check_std_symbols)
fi
if [[ -n "${LLGO_TEST_MODFILE:-}" ]]; then
  args+=("-modfile" "${LLGO_TEST_MODFILE}")
fi

printf '+'
printf ' %q' "${check_command[@]}"
for arg in "${args[@]}"; do
  printf ' %q' "${arg}"
done
printf '\n'

"${check_command[@]}" "${args[@]}"
