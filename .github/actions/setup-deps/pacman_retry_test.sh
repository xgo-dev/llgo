#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.github/actions/setup-deps/pacman_retry.sh
source "${script_dir}/pacman_retry.sh"

fail() {
	echo "$1" >&2
	exit 1
}

attempts=0
sleeps=0
last_sleep=""
last_args=()

# Called indirectly by pacman_with_retry.
# shellcheck disable=SC2329
pacman_command() {
	attempts=$((attempts + 1))
	last_args=("$@")
	(( attempts >= 3 ))
}

# Called indirectly by pacman_with_retry.
# shellcheck disable=SC2329
pacman_retry_sleep() {
	sleeps=$((sleeps + 1))
	last_sleep="$1"
}

LLGO_PACMAN_MAX_ATTEMPTS=3 LLGO_PACMAN_RETRY_DELAY_SECONDS=7 \
	pacman_with_retry --noconfirm -S --needed example-package 2>/dev/null
[[ "${attempts}" -eq 3 ]] || fail "pacman attempts = ${attempts}, want 3"
[[ "${sleeps}" -eq 2 ]] || fail "retry sleeps = ${sleeps}, want 2"
[[ "${last_sleep}" == 7 ]] || fail "retry delay = ${last_sleep}, want 7"
[[ " ${last_args[*]} " == *" --noconfirm -S --needed example-package "* ]] ||
	fail "unexpected pacman arguments: ${last_args[*]}"

attempts=0
sleeps=0
# Called indirectly by pacman_with_retry.
# shellcheck disable=SC2329
pacman_command() {
	attempts=$((attempts + 1))
	return 1
}

if LLGO_PACMAN_MAX_ATTEMPTS=2 LLGO_PACMAN_RETRY_DELAY_SECONDS=0 \
	pacman_with_retry -U package.pkg.tar.zst 2>/dev/null; then
	fail "exhausted pacman retries unexpectedly succeeded"
fi
[[ "${attempts}" -eq 2 ]] || fail "exhausted attempts = ${attempts}, want 2"
[[ "${sleeps}" -eq 1 ]] || fail "exhausted retry sleeps = ${sleeps}, want 1"

if LLGO_PACMAN_MAX_ATTEMPTS=0 pacman_with_retry -S package 2>/dev/null; then
	fail "invalid attempt count unexpectedly succeeded"
fi
if LLGO_PACMAN_RETRY_DELAY_SECONDS=invalid pacman_with_retry -S package 2>/dev/null; then
	fail "invalid retry delay unexpectedly succeeded"
fi
if pacman_with_retry 2>/dev/null; then
	fail "empty pacman command unexpectedly succeeded"
fi

echo "MSYS2 pacman retry checks passed"
