#!/usr/bin/env bash
set -euo pipefail

target_root=$PWD
compiler="$RUNNER_TEMP/llgo-bin/llgo"
binary="$DIAGNOSTIC_OUTPUT/weak-old.test"
baseline_parent=$(mktemp -d "$RUNNER_TEMP/weak-baseline.XXXXXX")
baseline_root="$baseline_parent/source"
test_group=
sampler=
cleanup() {
  if [[ -n "$test_group" ]]; then kill -- "-$test_group" 2>/dev/null || true; fi
  if [[ -n "$sampler" ]]; then kill "$sampler" 2>/dev/null || true; fi
  if [[ -d "$baseline_root" ]]; then
    git -C "$target_root" worktree remove "$baseline_root" || true
  fi
  rmdir "$baseline_parent" 2>/dev/null || true
}
trap cleanup EXIT

test "$(git rev-parse HEAD)" = "$DIAGNOSTIC_REVISION"
git fetch --no-tags --depth=1 origin "$DIAGNOSTIC_BASE_REVISION"
git worktree add --detach "$baseline_root" "$DIAGNOSTIC_BASE_REVISION"
test "$(git -C "$baseline_root" rev-parse HEAD)" = "$DIAGNOSTIC_BASE_REVISION"
printf 'Same compiler: %s\nTarget test source: %s/test/_stress/runtime/weak\nOld runtime source: %s/runtime\n' "$compiler" "$target_root" "$baseline_root"
printf 'The compiler and public stress test stay fixed; only LLGO_ROOT selects the pinned, unmodified baseline runtime. Package cache is disabled for this baseline build.\n'
git -C "$baseline_root" show -s --format='baseline commit=%H tree=%T'
git hash-object "$target_root/test/_stress/runtime/weak/weak_stress_test.go"

# Compile errors stop the step here. They can never satisfy the expected
# runtime-failure check below, and the tested target checkout is not modified.
cd "$target_root/test/_stress"
LLGO_ROOT="$baseline_root" GOWORK=off LLGO_BUILD_CACHE=off \
  "$compiler" test -c -o "$binary" ./runtime/weak 2>&1 | tee "$DIAGNOSTIC_OUTPUT/weak-old-build.log"
test -x "$binary"
binary=$(readlink -f "$binary")

# The public test has a 60s subprocess guard; the 80s process-group deadline
# also protects CI if the old runtime cannot service that parent-side guard.
unset LLGO_STRESS_WEAK_CHILD
setsid timeout --signal=TERM --kill-after=5s 80s \
  "$binary" -test.v -test.count=1 -test.timeout=3m \
  >"$DIAGNOSTIC_OUTPUT/weak-old-run.log" 2>&1 &
test_group=$!
(
  sleep 10
  sampled=0
  for pid in $(pgrep -g "$test_group" || true); do
    [[ "$(readlink "/proc/$pid/exe" 2>/dev/null || true)" == "$binary" ]] || continue
    # Inspect only the explicit child marker, never log the runner environment.
    if ! awk 'BEGIN { RS = "\0" } $0 == "LLGO_STRESS_WEAK_CHILD=TestWeakCleanupReentrancy" { found = 1 } END { exit !found }' \
      "/proc/$pid/environ"; then continue; fi
    ps -L -p "$pid" -o pid,tid,pcpu,stat,wchan:32,comm \
      >"$DIAGNOSTIC_OUTPUT/weak-old-threads-$pid.txt"
    printf 'Sampling the old-runtime stress child PID %s after 10s.\n' "$pid"
    # This intentionally perturbs only the expected-failing baseline, never
    # either fixed-runtime success trial. Disable remote debuginfo downloads.
    # shellcheck disable=SC2024
    sudo timeout --kill-after=2s 15s gdb --batch --nx -p "$pid" \
      -ex 'set pagination off' -ex 'set debuginfod enabled off' \
      -ex 'thread apply all bt 60' -ex detach \
      >"$DIAGNOSTIC_OUTPUT/weak-old-stack-$pid.txt" 2>&1 || true
    sampled=$((sampled + 1))
  done
  if [[ "$sampled" == 0 ]]; then
    echo 'No old-runtime stress child remained available for the 10s native stack sample.' >&2
    exit 1
  fi
) &
sampler=$!
status=0
wait "$test_group" || status=$?
sampler_status=0
wait "$sampler" || sampler_status=$?
sampler=
cat "$DIAGNOSTIC_OUTPUT/weak-old-run.log"
printf 'Old runtime exit=%s, sampler exit=%s.\n' "$status" "$sampler_status"

# A timeout, compiler error, assertion failure, or arbitrary panic is not a
# reproduction. Require the public test's guarded failure plus one native
# thread containing the exact recursive weak cleanup / map allocation chain.
signature=0
for stack in "$DIAGNOSTIC_OUTPUT"/weak-old-stack-*.txt; do
  [[ -f "$stack" ]] || continue
  if awk '
    function matches() { return callbacks >= 2 && collectors >= 2 && mapaccess && mutex && weakstate }
    function finish() { if (matches()) { print block; found = 1 } }
    /^Thread [0-9]+ / {
      finish()
      block = ""; callbacks = 0; collectors = 0; mapaccess = 0; mutex = 0; weakstate = 0
    }
    {
      block = block $0 "\n"
      if ($0 ~ /llgoRegisterWeakPointer[$]1/) callbacks++
      if ($0 ~ /GC_invoke_finalizers/) collectors++
      if ($0 ~ /Map(Access1|Delete)Fast64|map(access1|delete)_fast64/) mapaccess = 1
      if ($0 ~ /pthread_mutex_lock|lll_mutex_lock|lll_lock_wait/) mutex = 1
      if ($0 ~ /weakState/) weakstate = 1
    }
    END { finish(); exit !found }
  ' "$stack" >"$stack.signature.txt"; then signature=1; fi
done
if [[ "$status" != 1 || "$sampler_status" != 0 || "$signature" != 1 ]] || \
  ! grep -q -- '--- FAIL: TestWeakCleanupReentrancy' "$DIAGNOSTIC_OUTPUT/weak-old-run.log" || \
  ! grep -Eq 'weak cleanup helper (did not exit within 60s|failed)' "$DIAGNOSTIC_OUTPUT/weak-old-run.log"; then
  echo 'Baseline did not establish the expected guarded failure and same-thread weak cleanup self-deadlock signature; this is not a passing reproduction.' >&2
  exit 1
fi
printf '\nOld runtime: public weak stress failed as expected; native stack confirmed nested weak cleanup, map allocation, GC finalizer reentry, and weakState mutex self-lock on one thread.\n' >> "$GITHUB_STEP_SUMMARY"
