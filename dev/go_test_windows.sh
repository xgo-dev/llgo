#!/usr/bin/env bash

set -euo pipefail

if [[ $# -eq 0 ]]; then
	echo "usage: $0 <go-test-command> [argument ...]" >&2
	exit 2
fi

# Some GitHub-hosted Windows/amd64 machines are affected by golang/go#81238:
# recovering a hardware exception can write below a goroutine stack and
# corrupt an unrelated heap object. A corrupted testing.T signal channel then
# produces this otherwise-impossible synctest fatal error. This wrapper is used
# only for the host-test batch containing this module's test package tree.
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

log=
trap '[[ -z "${log}" ]] || rm -f "${log}"' EXIT
log="$(mktemp "${TMPDIR:-/tmp}/llgo-go-test-windows.XXXXXX")"
set +e
python "$(dirname "$0")/run_logged_command.py" "${log}" "$@"
status=$?
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
