#!/usr/bin/env bash
set -euo pipefail

# Linux-only continuation of .github/workflows/llgo.yml:jobs.llgo, after
# "Run Go 1.27 LLGo tests". Reuse the tested tree's existing scripts and keep
# each original step in a subshell so environment/cwd changes do not leak.
# This temporary harness belongs only to draft #2573, not either runtime PR.
test "$(git rev-parse HEAD)" = "$DIAGNOSTIC_REVISION"

run_step() {
  local name=$1 started=$SECONDS status=0
  shift
  printf '::group::%s\n' "$name"
  ( "$@" ) || status=$?
  printf '::endgroup::\n'
  printf '%s: exit=%s elapsed=%ss\n' "$name" "$status" "$((SECONDS - started))"
  printf '\n%s: exit=%s, %ss.\n' "$name" "$status" "$((SECONDS - started))" >> "$GITHUB_STEP_SUMMARY"
  return "$status"
}

# Expand mod_version in the child shell, not while constructing its command.
# shellcheck disable=SC2016
run_step 'Test Hello World with go.mod 1.21 through 1.26' bash -euo pipefail -c '
  for mod_version in 1.21 1.22 1.23 1.24 1.25 1.26; do
    LLGO_HELLO_EMBED=false dev/with_go_version.sh 1.27 dev/test_helloworld.sh "$mod_version"
  done
'
run_step 'Test Hello World with go.mod 1.27' dev/with_go_version.sh 1.27 dev/test_helloworld.sh 1.27
run_step 'Test runtime module compatibility' dev/test_runtime_go_versions.sh
run_step 'Test demo without RPATH (expect failure)' bash -euo pipefail -c '
  export LLGO_FULL_RPATH=false
  pkg-config --libs cargs
  if (cd ./_demo/c/cargs && llgo run .); then
    echo "ERROR: cargs demo should have failed without RPATH!"
    exit 1
  else
    echo "cargs demo correctly failed without RPATH"
  fi
'
run_step 'Test demos' bash .github/workflows/test_demo.sh
run_step 'Test demos (embedded target)' bash .github/workflows/test_demo.sh --embedded
run_step 'Test C header generation' bash -euo pipefail -c '
  cd _demo/go/export
  chmod +x test.sh
  ./test.sh
'
run_step 'Test export with different symbol names on embedded targets' bash -euo pipefail -c '
  cd _demo/embed/export
  chmod +x verify_export.sh
  ./verify_export.sh
'
run_step 'Test ESP serial smoke' bash -euo pipefail -c '
  cd _demo/embed
  chmod +x test-esp-serial-startup.sh
  ./test-esp-serial-startup.sh
'
run_step 'Test ESP32-C3 startup regression' bash -euo pipefail -c '
  python -m pip install esptool==5.1.0
  cd _demo/embed
  chmod +x test_esp32c3_startup.sh
  ./test_esp32c3_startup.sh
'
run_step '_xtool build tests' bash -euo pipefail -c '
  cd _xtool
  llgo build -v ./...
'
run_step 'Show test result' cat result.md
run_step 'Install LLDB for integration tests' sudo apt-get install -y lldb-22
run_step 'LLDB integration tests' bash cmd/llgo/lldbtest/runtest.sh -v
printf '\nAll original Linux shard-0 job integration checks passed.\n' >> "$GITHUB_STEP_SUMMARY"
