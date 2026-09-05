#!/usr/bin/env bash

# LLGO_PACMAN_MAX_ATTEMPTS: total transaction attempts (default: 3).
# LLGO_PACMAN_RETRY_DELAY_SECONDS: delay between attempts (default: 5).
# Retries intentionally accept any pacman failure, including permanent errors;
# the bounded attempt count limits the delay before reporting exhaustion.

pacman_command() {
	command pacman "$@"
}

pacman_retry_sleep() {
	command sleep "$1"
}

pacman_with_retry() {
	local max_attempts="${LLGO_PACMAN_MAX_ATTEMPTS:-3}"
	local retry_delay="${LLGO_PACMAN_RETRY_DELAY_SECONDS:-5}"
	if ! [[ "${max_attempts}" =~ ^[1-9][0-9]*$ ]]; then
		echo "LLGO_PACMAN_MAX_ATTEMPTS must be a positive integer" >&2
		return 2
	fi
	if ! [[ "${retry_delay}" =~ ^[0-9]+$ ]]; then
		echo "LLGO_PACMAN_RETRY_DELAY_SECONDS must be a non-negative integer" >&2
		return 2
	fi
	if (( $# == 0 )); then
		echo "no pacman arguments specified" >&2
		return 2
	fi

	local attempt
	for ((attempt = 1; attempt <= max_attempts; attempt++)); do
		if pacman_command "$@"; then
			return 0
		fi
		if (( attempt < max_attempts )); then
			echo "pacman failed (attempt ${attempt}/${max_attempts}); retrying the package transaction" >&2
			pacman_retry_sleep "${retry_delay}"
		fi
	done

	echo "pacman failed after ${max_attempts} attempts" >&2
	return 1
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	set -euo pipefail
	pacman_with_retry "$@"
fi
