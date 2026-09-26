#!/usr/bin/env bash

# Shared accounting for the WebAssembly runtime acceptance scripts. Each case
# writes one append-only record so an EXIT trap can still report the last
# failure after errexit stops the suite.

wasm_ci_report_init() {
	WASM_CI_REPORT_FILE="$1"
	: >"${WASM_CI_REPORT_FILE}"
}

wasm_ci_run_case() {
	local profile="$1"
	local case_name="$2"
	local executed_packages="$3"
	local expected_failures="$4"
	local timeouts="$5"
	local skipped="$6"
	local inapplicable="$7"
	shift 7

	local restore_errexit=0
	case "$-" in
	*e*) restore_errexit=1 ;;
	esac
	set +e
	(
		set -e
		"$@"
	)
	local status=$?
	if [[ ${restore_errexit} -eq 1 ]]; then
		set -e
	else
		set +e
	fi

	local unexpected_failure=0
	if [[ ${status} -ne 0 ]]; then
		unexpected_failure=1
		# The case did not reach its acceptance boundary, so planned coverage
		# must not be reported as executed coverage.
		executed_packages=0
		expected_failures=0
		timeouts=0
		skipped=0
		inapplicable=0
	fi
	printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
		"${profile}" "${case_name}" "${executed_packages}" \
		"${expected_failures}" "${timeouts}" "${skipped}" \
		"${inapplicable}" "${unexpected_failure}" >>"${WASM_CI_REPORT_FILE}"
	return "${status}"
}

wasm_ci_render_report() {
	local report_file="$1"
	local suite_status="$2"
	awk -F '\t' -v suite_status="${suite_status}" '
		BEGIN {
			print "### WebAssembly single-worker coverage"
			print ""
			print "| Profile | Checks | Executed package runs | Expected failure paths | Timeout paths | Unexpected failures | Skipped | Inapplicable |"
			print "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |"
		}
		{
			profile = $1
			if (!seen[profile]++) {
				order[++profiles] = profile
			}
			checks[profile]++
			packages[profile] += $3
			expected[profile] += $4
			timeouts[profile] += $5
			skipped[profile] += $6
			inapplicable[profile] += $7
			unexpected[profile] += $8
			total_checks++
			total_packages += $3
			total_expected += $4
			total_timeouts += $5
			total_skipped += $6
			total_inapplicable += $7
			total_unexpected += $8
			if ($8 != 0) {
				failed[++failures] = profile "/" $2
			}
		}
		END {
			for (i = 1; i <= profiles; i++) {
				profile = order[i]
				printf "| %s | %d | %d | %d | %d | %d | %d | %d |\n", profile, checks[profile], packages[profile], expected[profile], timeouts[profile], unexpected[profile], skipped[profile], inapplicable[profile]
			}
			printf "| **Total** | **%d** | **%d** | **%d** | **%d** | **%d** | **%d** | **%d** |\n", total_checks, total_packages, total_expected, total_timeouts, total_unexpected, total_skipped, total_inapplicable
			print ""
			if (suite_status == 0) {
				print "Suite result: pass."
			} else {
				printf "Suite result: fail (exit status %d).\n", suite_status
				for (i = 1; i <= failures; i++) {
					printf "- Unexpected failure: `%s`\n", failed[i]
				}
			}
		}
	' "${report_file}"
}

wasm_ci_publish_report() {
	local report_file="$1"
	local suite_status="$2"
	local rendered
	rendered="$(wasm_ci_render_report "${report_file}" "${suite_status}")"
	printf '%s\n' "${rendered}"
	if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
		printf '%s\n' "${rendered}" >>"${GITHUB_STEP_SUMMARY}"
	fi
}
