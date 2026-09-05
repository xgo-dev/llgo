#!/usr/bin/env bash

set -euo pipefail

if [[ $# -eq 0 ]]; then
	echo "usage: $0 <go-test-command> [argument ...]" >&2
	echo "       $0 --retry-test-go [coverage flags] ./test/go" >&2
	exit 2
fi

# By default this wraps the host-test batch containing this module's test tree.
# --retry-test-go is a separate, opt-in mode for the Windows MinGW x64 coverage
# command only. It retries an observed access violation, not a confirmed runtime
# bug, and requires a successful full package run instead of quarantining it.
#
# Some GitHub-hosted Windows/amd64 machines are affected by golang/go#81238:
# recovering a hardware exception can write below a goroutine stack and
# corrupt an unrelated heap object. A corrupted testing.T signal channel then
# produces this otherwise-impossible synctest fatal error.
module_path="$(go list -m -f '{{.Path}}')"

is_known_runtime_corruption() {
	local log="$1"
	# The runtime provides no machine-readable cause. Require one exact fatal
	# plus only the expected package failure, and reject other failure markers.
	awk '
		BEGIN { want = "fatal error: receive on synctest channel from outside bubble" }
		{
			line = $0
			sub(/\r$/, "", line)
			if (index(line, "fatal error:") == 1) {
				count++
				if (line != want) other = 1
			}
		}
		END { exit !(count == 1 && !other) }
	' "${log}" &&
		grep -Fq 'testing.(*T).Run' "${log}" &&
		awk -v prefix="${module_path}/test" '
			$1 == "FAIL" && NF >= 2 {
				if ($2 == prefix || index($2, prefix "/") == 1) target = 1
				else other = 1
			}
			END { exit !(target && !other) }
		' "${log}" &&
		! grep -Eiq '^--- FAIL:|\[build failed\]|^panic:|WARNING: DATA RACE|test timed out|SIGQUIT|SIGSEGV|SIGABRT|unexpected fault address|signal: (segmentation fault|aborted)' "${log}"
}

is_test_go_access_violation() {
	local log="$1"
	awk '
		{ sub(/\r$/, "") }
		/^exit status / {
			if ($0 == "exit status 0xc0000005") count++
			else other = 1
		}
		END { exit !(count == 1 && !other) }
	' "${log}" &&
		awk -v target="${module_path}/test/go" '
			$1 == "FAIL" && NF >= 2 {
				if ($2 == target) target_failure++
				else other_failure = 1
			}
			END { exit !(target_failure == 1 && !other_failure) }
		' "${log}" &&
		! grep -Eiq '^[[:space:]]*--- FAIL:|\[build failed\]|^fatal error:|^panic:|WARNING: DATA RACE|test timed out|SIGQUIT|SIGSEGV|SIGABRT|unexpected fault address|signal: (segmentation fault|aborted)' "${log}"
}

report_test_go_retry() {
	echo "LLGO_CI_WINDOWS_TEST_GO_RETRY: $*"
	if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
		printf -- '- Windows test/go retry: %s\n' "$*" >>"${GITHUB_STEP_SUMMARY}"
	fi
}

retry_test_go() {
	if [[ "${RUNNER_OS:-}" != Windows || "${RUNNER_ARCH:-}" != X64 ]]; then
		echo '--retry-test-go is only supported on Windows x64 CI runners' >&2
		return 2
	fi

	# This is a workflow-specific API, not an arbitrary command wrapper. Accept
	# only its coverage flags and one test/go package, so another package or a
	# flag such as -c, -list, -run or -args cannot bypass a full test execution.
	local arg target_count=0
	local command=(go test)
	for arg in "$@"; do
		case "${arg}" in
		-ldflags=*|-timeout=*|-coverprofile=*|-covermode=*)
			command+=("${arg}")
			;;
		./test/go|"${module_path}/test/go")
			target_count=$((target_count + 1))
			;;
		*)
			echo "unsupported --retry-test-go argument: ${arg}" >&2
			return 2
			;;
		esac
	done
	if [[ ${target_count} -ne 1 ]]; then
		echo '--retry-test-go requires exactly one ./test/go package' >&2
		return 2
	fi
	# Disable test-result caching, not compilation caching. Each attempt starts
	# a new go test process; -count=3 would repeat inside the same test process.
	command+=(-count=1 ./test/go)

	local attempt status
	local statuses=()
	for ((attempt = 1; attempt <= 3; attempt++)); do
		report_test_go_retry "Starting attempt ${attempt}/3 in a new process: all tests in ./test/go, -count=1 (no cached test results). Other packages and CI steps are not rerun."
		printf 'Command:'
		printf ' %q' "${command[@]}"
		printf '\n'
		set +e
		"${command[@]}" 2>&1 | tee "${log}"
		statuses=("${PIPESTATUS[@]}")
		set -e
		status=${statuses[0]}
		if [[ ${statuses[1]} -ne 0 ]]; then
			report_test_go_retry "Attempt ${attempt}/3: log capture failed; failing without retry."
			return 1
		fi
		if [[ ${status} -eq 0 ]]; then
			report_test_go_retry "Attempt ${attempt}/3 passed. Tests completed successfully; this retry does not disable coverage upload."
			return 0
		fi
		if ! is_test_go_access_violation "${log}"; then
			report_test_go_retry "Attempt ${attempt}/3 failed (exit ${status}) without the eligible test/go-only 0xc0000005 signature; failing without further retries."
			return "${status}"
		fi
		if [[ ${attempt} -eq 3 ]]; then
			report_test_go_retry "Attempt 3/3 failed with 0xc0000005; both retries exhausted. No successful package run; failing (exit ${status}), not quarantining."
			return "${status}"
		fi
		echo '::warning title=Windows test/go access violation::Retrying the standalone test/go package after 0xc0000005; the root cause is unconfirmed and a successful run is required.'
		report_test_go_retry "Attempt ${attempt}/3 failed (exit ${status}) with the observed test/go-only 0xc0000005 signature. Root cause unconfirmed; retrying the whole package in a new process. Original output is retained above."
	done
}

log=
trap '[[ -z "${log}" ]] || rm -f "${log}"' EXIT
log="$(mktemp "${TMPDIR:-/tmp}/llgo-go-test-windows.XXXXXX")"
if [[ "$1" == --retry-test-go ]]; then
	shift
	retry_test_go "$@"
	exit 0
fi

set +e
"$@" 2>&1 | tee "${log}"
status=${PIPESTATUS[0]}
set -e

if [[ ${status} -eq 0 ]] || ! is_known_runtime_corruption "${log}"; then
	exit "${status}"
fi

echo '::warning title=Quarantined upstream Go runtime corruption::LLGO_CI_QUARANTINED_GO_RUNTIME_CORRUPTION: matched the narrow golang/go#81238 signature'
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
	echo 'windows_runtime_corruption=true' >>"${GITHUB_OUTPUT}"
fi
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
	{
		echo '### Quarantined Windows Go runtime corruption'
		echo
		echo 'The narrow golang/go#81238 signature occurred in the LLGo test package tree. The failure was quarantined and the coverage upload for this job was skipped.'
	} >>"${GITHUB_STEP_SUMMARY}"
fi
exit 0
