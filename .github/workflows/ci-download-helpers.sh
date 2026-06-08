#!/usr/bin/env bash

ci_is_external_download_failure() {
  local log_file="$1"
  grep -Eq \
    'failed to setup crosscompile|bad status: (5[0-9][0-9]|429)|Gateway Timeout|download failed .*retrying|ESP QEMU download failed|Failed to download .* after retries|curl: \([0-9]+\).*(error: 5|CONNECT tunnel failed|Empty reply)' \
    "$log_file"
}

ci_run_optional_download_test() {
  local name="$1"
  shift

  local log_dir="${RUNNER_TEMP:-/tmp}"
  local log_file
  log_file="$(mktemp "${log_dir}/optional-download-test.XXXXXX.log")"

  CI_OPTIONAL_DOWNLOAD_TEST_SKIPPED=0
  local errexit_state
  errexit_state="$(set -o | awk '$1 == "errexit" { print $2 }')"
  set +e
  "$@" 2>&1 | tee "$log_file"
  local status=${PIPESTATUS[0]}
  if [ "$errexit_state" = "on" ]; then
    set -e
  else
    set +e
  fi

  if [ "$status" -eq 0 ]; then
    return 0
  fi
  if ci_is_external_download_failure "$log_file"; then
    CI_OPTIONAL_DOWNLOAD_TEST_SKIPPED=1
    echo "::warning::${name} skipped because an external CI download is unavailable"
    return 0
  fi
  return "$status"
}
