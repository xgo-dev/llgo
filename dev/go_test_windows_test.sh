#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

LLGO_TEST_MODULE_PATH="$(go list -m -f '{{.Path}}')"
export LLGO_TEST_MODULE_PATH
test_root="$(mktemp -d)"
trap 'rm -rf "${test_root}"' EXIT
case_count=0

# Export a fake Go command into each wrapper process. State lives on disk so
# separate attempts see the next result without executing real Windows tests.
go() {
	if [[ "$1" == list ]]; then
		printf '%s\n' "${LLGO_TEST_MODULE_PATH}"
		return 0
	fi
	[[ "$1" == test ]] || return 99
	local count=0 stages=() stage eol=$'\n'
	if [[ -f "${LLGO_TEST_CASE}/count" ]]; then
		read -r count <"${LLGO_TEST_CASE}/count"
	fi
	count=$((count + 1))
	printf '%s\n' "${count}" >"${LLGO_TEST_CASE}/count"
	printf '%s\n' "$@" >>"${LLGO_TEST_CASE}/args"
	read -r -a stages <<<"${LLGO_TEST_SEQUENCE}"
	stage="${stages[count - 1]:-unexpected}"
	printf 'Fake go test attempt %s: %s\n' "${count}" "${stage}"
	case "${stage}" in
	pass)
		printf 'ok\t%s/test/go\t0.1s\n' "${LLGO_TEST_MODULE_PATH}"
		return 0
		;;
	av)
		[[ "${LLGO_TEST_CRLF}" == 0 ]] || eol=$'\r\n'
		printf 'exit status 0xc0000005%sFAIL\t%s/%s\t0.1s%s' \
			"${eol}" "${LLGO_TEST_MODULE_PATH}" "${LLGO_TEST_PACKAGE}" "${eol}"
		printf '%s' "${LLGO_TEST_EXTRA}"
		return 23
		;;
	fail)
		printf -- '--- FAIL: TestRealFailure\nFAIL\t%s/test/go\t0.1s\n' "${LLGO_TEST_MODULE_PATH}"
		return 17
		;;
	synctest)
		printf 'fatal error: receive on synctest channel from outside bubble\ntesting.(*T).Run\nFAIL\t%s/%s\t0.1s\n' \
			"${LLGO_TEST_MODULE_PATH}" "${LLGO_TEST_PACKAGE}"
		return 31
		;;
	*) return 99 ;;
	esac
}
export -f go

fail() {
	echo "${LLGO_TEST_CASE}: $*" >&2
	if [[ -f "${LLGO_TEST_CASE}/log" ]]; then
		sed -n '1,100p' "${LLGO_TEST_CASE}/log" >&2
	fi
	exit 1
}

new_case() {
	case_count=$((case_count + 1))
	export LLGO_TEST_CASE="${test_root}/${case_count}-$1"
	mkdir "${LLGO_TEST_CASE}"
	export LLGO_TEST_SEQUENCE="$2" LLGO_TEST_EXTRA='' LLGO_TEST_CRLF=0 LLGO_TEST_PACKAGE=test/go
	export RUNNER_OS=Windows RUNNER_ARCH=X64
	export GITHUB_OUTPUT="${LLGO_TEST_CASE}/output" GITHUB_STEP_SUMMARY="${LLGO_TEST_CASE}/summary"
	: >"${GITHUB_OUTPUT}"
	: >"${GITHUB_STEP_SUMMARY}"
}

run_case() {
	local want_count="$1" want_status="$2" status=0 count=0
	shift 2
	if bash dev/go_test_windows.sh "$@" >"${LLGO_TEST_CASE}/log" 2>&1; then
		status=0
	else
		status=$?
	fi
	[[ "${status}" == "${want_status}" ]] || fail "exit ${status}, expected ${want_status}"
	if [[ -f "${LLGO_TEST_CASE}/count" ]]; then
		read -r count <"${LLGO_TEST_CASE}/count"
	fi
	[[ "${count}" == "${want_count}" ]] || fail "ran ${count} times, expected ${want_count}"
}

no_coverage_skip() {
	[[ ! -s "${GITHUB_OUTPUT}" ]] || fail 'unexpected coverage-skip output'
}

has_log() {
	grep -Fq -- "$1" "${LLGO_TEST_CASE}/log" || fail "missing log: $1"
	grep -Fq -- "$1" "${GITHUB_STEP_SUMMARY}" || fail "missing summary: $1"
}

new_case first_pass pass
run_case 1 0 --retry-test-go -timeout=45m \
	'-ldflags=-linkmode=external -extldflags=-lsynchronization' \
	"-coverprofile=${LLGO_TEST_CASE}/coverage result.txt" -covermode=atomic ./test/go
for arg in test -timeout=45m '-ldflags=-linkmode=external -extldflags=-lsynchronization' \
	"-coverprofile=${LLGO_TEST_CASE}/coverage result.txt" -covermode=atomic -count=1 ./test/go; do
	grep -Fxq -- "${arg}" "${LLGO_TEST_CASE}/args" || fail "lost or split argument: ${arg}"
done
[[ "$(wc -l <"${LLGO_TEST_CASE}/args")" -eq 7 ]] || fail 'unexpected test arguments'
has_log 'Starting attempt 1/3 in a new process: all tests in ./test/go, -count=1'
has_log 'Other packages and CI steps are not rerun.'
has_log 'Attempt 1/3 passed.'
no_coverage_skip

new_case retry_once 'av pass'
run_case 2 0 --retry-test-go ./test/go
has_log 'Root cause unconfirmed; retrying the whole package in a new process.'
has_log 'Original output is retained above.'
has_log 'Attempt 2/3 passed.'
grep -Fq 'exit status 0xc0000005' "${LLGO_TEST_CASE}/log" || fail 'original crash output lost'
[[ "$(grep -Fxc -- '-count=1' "${LLGO_TEST_CASE}/args")" -eq 2 ]] || fail 'a retry could use cached test results'
no_coverage_skip

new_case retry_twice 'av av pass'
run_case 3 0 --retry-test-go ./test/go
has_log 'Attempt 3/3 passed.'
no_coverage_skip

new_case exhausted 'av av av pass'
run_case 3 23 --retry-test-go ./test/go
has_log 'both retries exhausted. No successful package run; failing'
no_coverage_skip

new_case real_failure_after_crash 'av fail pass'
run_case 2 17 --retry-test-go ./test/go
has_log 'failing without further retries.'
no_coverage_skip

new_case crlf 'av pass'
export LLGO_TEST_CRLF=1
run_case 2 0 --retry-test-go ./test/go
has_log 'Attempt 2/3 passed.'
no_coverage_skip

new_case module_qualified pass
run_case 1 0 --retry-test-go "${LLGO_TEST_MODULE_PATH}/test/go"
grep -Fxq './test/go' "${LLGO_TEST_CASE}/args" || fail 'package was not normalized'
no_coverage_skip

new_case no_opt_in 'av pass'
run_case 1 23 go test ./test/go
no_coverage_skip
if grep -Fq 'LLGO_CI_WINDOWS_TEST_GO_RETRY' "${LLGO_TEST_CASE}/log"; then
	fail 'default wrapper unexpectedly retried an access violation'
fi

for platform in Linux macOS Windows; do
	new_case wrong_platform pass
	export RUNNER_OS="${platform}"
	[[ "${platform}" != Windows ]] || export RUNNER_ARCH=ARM64
	run_case 0 2 --retry-test-go ./test/go
	no_coverage_skip
done

new_case missing_package pass
run_case 0 2 --retry-test-go
no_coverage_skip
for arg in ./cl ./test net/http "${LLGO_TEST_MODULE_PATH}/cl" ./test/go \
	"${LLGO_TEST_MODULE_PATH}/test/go" -c -list=. -run=. -skip=. -args -- -count=3 -timeout; do
	new_case unsupported_argument pass
	run_case 0 2 --retry-test-go "${arg}" ./test/go
	no_coverage_skip
done

# No ordinary failure or additional diagnostic may turn into a retry, even
# when the access-violation line is also present and a later attempt would pass.
for diagnostic in \
	$'--- FAIL: TestRealFailure\n' $'    --- FAIL: TestNestedFailure\n' \
	$'[build failed]\n' $'fatal error: other failure\n' $'panic: failure\n' \
	$'WARNING: DATA RACE\n' $'test timed out\n' $'SIGQUIT\n' $'SIGSEGV\n' \
	$'SIGABRT\n' $'unexpected fault address\n' $'signal: segmentation fault\n' \
	$'signal: aborted\n' $'exit status 0xc0000005\n' $'exit status 2\n' \
	"$(printf 'FAIL\t%s/cl\t0.1s\n' "${LLGO_TEST_MODULE_PATH}")" \
	"$(printf 'FAIL\t%s/test/go\t0.1s\n' "${LLGO_TEST_MODULE_PATH}")"; do
	new_case ineligible_diagnostic 'av pass'
	export LLGO_TEST_EXTRA="${diagnostic}"
	run_case 1 23 --retry-test-go ./test/go
	no_coverage_skip
done

new_case wrong_failure_package 'av pass'
export LLGO_TEST_PACKAGE=cl
run_case 1 23 --retry-test-go ./test/go
no_coverage_skip

new_case ordinary_failure 'fail pass'
run_case 1 17 --retry-test-go ./test/go
no_coverage_skip

new_case synctest_not_retryable 'synctest pass'
run_case 1 31 --retry-test-go ./test/go
no_coverage_skip

new_case legacy_synctest synctest
export LLGO_TEST_PACKAGE=test
run_case 1 0 go test "${LLGO_TEST_MODULE_PATH}/test"
grep -Fxq 'windows_runtime_corruption=true' "${GITHUB_OUTPUT}" || fail 'legacy quarantine lost coverage skip'
grep -Fq 'LLGO_CI_QUARANTINED_GO_RUNTIME_CORRUPTION' "${LLGO_TEST_CASE}/log" || fail 'legacy quarantine marker lost'

new_case preserve_legacy_skip 'av pass'
printf 'windows_runtime_corruption=true\n' >"${GITHUB_OUTPUT}"
run_case 2 0 --retry-test-go ./test/go
[[ "$(<"${GITHUB_OUTPUT}")" == windows_runtime_corruption=true ]] || fail 'retry changed an existing coverage-skip output'

printf 'Windows Go-test wrapper: %s cases passed\n' "${case_count}"
