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

run_goroot() {
	local goroot=$1
	go_bin="$goroot/bin/go"
	if [[ "${OS:-}" == "Windows_NT" ]]; then
		go_bin+=".exe"
	fi
	if [[ ! -x "$go_bin" ]]; then
		echo "error: missing go binary: $go_bin" >&2
		exit 2
	fi
	version="$("$go_bin" env GOVERSION)"
	echo "==== $version ($goroot) ===="
	(
		cd "$repo_root"
		if [[ -n "${LLGO_GOROOT_CACHE_DIR:-}" ]]; then
			version_cache_dir="${LLGO_GOROOT_CACHE_DIR}/${version}"
			mkdir -p "$version_cache_dir"
			export XDG_CACHE_HOME="$version_cache_dir"
		fi
		goroot_gomaxprocs="${LLGO_GOROOT_GOMAXPROCS:-${GOMAXPROCS:-2}}"
		# CI sets this below the enclosing job timeout so the Go runner can
		# print its timeout diagnostics and the workflow can upload its report.
		goroot_test_timeout="${LLGO_GOROOT_TEST_TIMEOUT:-180m}"
		if [[ -n "${LLGO_GOROOT_RUNNER:-}" ]]; then
			cd "$repo_root/test/goroot"
			test_args=(-test.run='^TestGoRootRunCases$' -test.count=1 -test.timeout="${goroot_test_timeout}")
			if [[ "${LLGO_GOROOT_VERBOSE:-0}" != "0" ]]; then
				test_args+=("-test.v")
			fi
			run_with_heartbeat env GOMAXPROCS="$goroot_gomaxprocs" \
				"${LLGO_GOROOT_RUNNER}" "${test_args[@]}" \
				-goroot "$goroot" "${runner_args[@]}"
		else
			go_test_args=()
			if [[ "${LLGO_GOROOT_VERBOSE:-0}" != "0" ]]; then
				go_test_args+=("-v")
			fi
			run_with_heartbeat env GOMAXPROCS="$goroot_gomaxprocs" \
				go test -p=1 ./test/goroot "${go_test_args[@]}" -run='^TestGoRootRunCases$' \
				-count=1 -timeout "$goroot_test_timeout" -args -goroot "$goroot" "${runner_args[@]}"
		fi
	)
}

if [[ "${LLGO_GOROOT_PARALLEL:-0}" != "0" && ${#goroots[@]} -gt 1 ]]; then
	if [[ "${LLGO_GOROOT_PARALLEL}" != "1" ]]; then
		echo "error: LLGO_GOROOT_PARALLEL must be 0 or 1" >&2
		exit 2
	fi
	log_dir="${LLGO_GOROOT_LOG_DIR:-}"
	if [[ -n "$log_dir" ]]; then
		mkdir -p "$log_dir"
	fi
	pids=()
	for goroot in "${goroots[@]}"; do
		go_bin="$goroot/bin/go"
		if [[ "${OS:-}" == "Windows_NT" ]]; then
			go_bin+=".exe"
		fi
		if [[ ! -x "$go_bin" ]]; then
			echo "error: missing go binary: $go_bin" >&2
			exit 2
		fi
		version="$("$go_bin" env GOVERSION)"
		(
			if [[ -z "$log_dir" ]]; then
				run_goroot "$goroot"
				exit
			fi
			set +e
			run_goroot "$goroot" 2>&1 | tee "$log_dir/${version}.log"
			pipe_status=("${PIPESTATUS[@]}")
			set -e
			if [[ ${pipe_status[0]} -ne 0 ]]; then
				exit "${pipe_status[0]}"
			fi
			exit "${pipe_status[1]}"
		) &
		pids+=("$!")
	done
	status=0
	for pid in "${pids[@]}"; do
		if ! wait "$pid"; then
			status=1
		fi
	done
	exit "$status"
fi

for goroot in "${goroots[@]}"; do
	run_goroot "$goroot"
done
