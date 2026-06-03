#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

runner_args=()
goroots=()
after_delim=0
for arg in "$@"; do
	if [[ "$arg" == "--" ]]; then
		after_delim=1
		continue
	fi
	if [[ $after_delim -eq 1 ]]; then
		runner_args+=("$arg")
	else
		goroots+=("$arg")
	fi
done

if [[ ${#goroots[@]} -eq 0 ]]; then
	if [[ -n "${LLGO_GOROOT_MATRIX:-}" ]]; then
		IFS=':' read -r -a goroots <<<"${LLGO_GOROOT_MATRIX}"
	else
		goroots=("$(go env GOROOT)")
	fi
fi

run_with_heartbeat() {
	local interval="${LLGO_GOROOT_HEARTBEAT_SECONDS:-0}"
	if [[ "$interval" == "0" ]]; then
		"$@"
		return
	fi
	if ! [[ "$interval" =~ ^[0-9]+$ ]] || [[ "$interval" -le 0 ]]; then
		echo "error: LLGO_GOROOT_HEARTBEAT_SECONDS must be a positive integer or 0" >&2
		exit 2
	fi

	"$@" &
	local cmd_pid=$!
	(
		local elapsed=0
		while sleep "$interval"; do
			if ! kill -0 "$cmd_pid" 2>/dev/null; then
				exit 0
			fi
			elapsed=$((elapsed + interval))
			echo "goroot runner still running (${elapsed}s elapsed)"
		done
	) &
	local heartbeat_pid=$!

	local status=0
	wait "$cmd_pid" || status=$?
	kill "$heartbeat_pid" 2>/dev/null || true
	wait "$heartbeat_pid" 2>/dev/null || true
	return "$status"
}

for goroot in "${goroots[@]}"; do
	go_bin="$goroot/bin/go"
	if [[ ! -x "$go_bin" ]]; then
		echo "error: missing go binary: $go_bin" >&2
		exit 2
	fi
	version="$("$go_bin" env GOVERSION)"
	echo "==== $version ($goroot) ===="
	(
		cd "$repo_root"
		go_test_args=()
		if [[ "${LLGO_GOROOT_VERBOSE:-0}" != "0" ]]; then
			go_test_args+=("-v")
		fi
		run_with_heartbeat go test ./test/goroot "${go_test_args[@]}" -count=1 -timeout 180m -args -goroot "$goroot" "${runner_args[@]}"
	)
done
