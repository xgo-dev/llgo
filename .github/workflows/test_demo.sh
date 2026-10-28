#!/bin/bash
set -e

mode="host"
case "${1:-}" in
  --embedded)
    mode="embedded"
    shift
    ;;
  --embedded-build-only)
    mode="embedded-build-only"
    shift
    ;;
esac

# Keep the scheduler deliberately small: directory consolidation owns coverage,
# while this script only selects the few specialized profiles and runs them.
jobs="${LLGO_DEMO_JOBS:-4}"
if [ "$mode" != "host" ] && [ -z "${LLGO_DEMO_JOBS:-}" ]; then
  # The first case for each target warms an expensive shared build cache. Run
  # embedded cases sequentially by default so that cold work happens once.
  jobs=1
fi
case "$jobs" in
  ""|*[!0-9]*|0)
    echo "LLGO_DEMO_JOBS must be a positive integer" >&2
    exit 2
    ;;
esac

llgo_run_flags=()
if [ -n "${LLGO_DEMO_LLGORUN_FLAGS:-}" ]; then
  read -r -a llgo_run_flags <<< "${LLGO_DEMO_LLGORUN_FLAGS}"
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
python="${LLGO_DEMO_PYTHON:-python3}"
if [ "${RUNNER_OS:-}" = "Windows" ] && [ -z "${LLGO_DEMO_PYTHON:-}" ]; then
  if [ -z "${LLGO_WINDOWS_PYTHON_HOME:-}" ]; then
    echo "LLGO_WINDOWS_PYTHON_HOME is required for native Windows ownership" >&2
    exit 2
  fi
  python="$(cygpath -u "$LLGO_WINDOWS_PYTHON_HOME")/python.exe"
fi
if ! command -v "$python" >/dev/null 2>&1; then
  echo "Python is required for owned demo stage timeouts" >&2
  exit 2
fi
"$python" -c 'import os,sys; print("Demo supervisor:",sys.executable,sys.version.split()[0],"owner="+os.name); sys.exit(2 if os.environ.get("RUNNER_OS")=="Windows" and os.name!="nt" else 0)'
build_timeout="${LLGO_DEMO_BUILD_TIMEOUT:-300}"
run_timeout="${LLGO_DEMO_RUN_TIMEOUT:-120}"
# Cold embedded toolchains include target runtime/SDK compilation. Keep their
# explicit budget independent of the short native executable run deadline.
embedded_timeout="${LLGO_DEMO_EMBEDDED_TIMEOUT:-1200}"
tmp_root="$(mktemp -d)"
if [ -n "${LLGO_DEMO_OUTPUT_DIR:-}" ]; then
  log_root="$LLGO_DEMO_OUTPUT_DIR/$mode"
  keep_logs=1
else
  log_root="$(mktemp -d)"
  keep_logs=0
fi
mkdir -p "$log_root"
log_root="$(cd "$log_root" && pwd)"
active_pids=()

cleanup() {
  local status=$?
  trap - EXIT INT TERM
  for pid in "${active_pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  for pid in "${active_pids[@]}"; do
    wait "$pid" 2>/dev/null || true
  done
  rm -rf "$tmp_root"
  if [ "$status" -eq 0 ] && [ "$keep_logs" -eq 0 ]; then
    rm -rf "$log_root"
  else
    echo "Demo stage logs retained in $log_root"
  fi
  exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

run_dirs=()
run_targets=()
run_labels=()
emulator=0

add_case() {
  run_dirs+=("$1")
  run_targets+=("$2")
  run_labels+=("$3")
}

if [ "$mode" != "host" ]; then
  if [ "$mode" = "embedded" ]; then
    emulator=1
  fi

  # Positive target lists avoid recursively treating support packages as demos.
  esp32_cases=(
    ./_demo/c/catomic
    ./_demo/c/hello
    ./_demo/c/qsort
    ./_demo/go/ifaceconv
    ./_demo/go/linkname
  )
  esp32c3_cases=(
    ./_demo/c/hello
    ./_demo/c/qsort
    ./_demo/go/cabi
    ./_demo/go/ifaceconv
    ./_demo/go/issue1538
    ./_demo/go/linkname
  )
  for d in "${esp32_cases[@]}"; do
    add_case "$d" esp32 "$d (target=esp32)"
  done
  for d in "${esp32c3_cases[@]}"; do
    add_case "$d" esp32c3-basic "$d (target=esp32c3-basic)"
  done
else
  cases=()
  use_lto=0
  use_globaldce=0
  use_deadcodedrop=0
  for flag in "${llgo_run_flags[@]}"; do
    case "$flag" in
      -lto=full) use_lto=1 ;;
      -globaldce) use_globaldce=1 ;;
      -deadcodedrop) use_deadcodedrop=1 ;;
    esac
  done

  if [ "$use_lto" -eq 1 ] && [ "$use_globaldce" -eq 1 ]; then
    cases=(
      ./_demo/c/asmfullcall
      ./_demo/c/cgofull
      ./_demo/c/cabisret
      ./_demo/c/cppintf
      ./_demo/go/cabi
      ./_demo/go/export
      ./_demo/go/reflect
      ./_demo/go/stdlib
      ./_demo/go/sync
    )
  elif [ "$use_deadcodedrop" -eq 1 ]; then
    cases=(
      ./_demo/c/asmfullcall
      ./_demo/c/cgofull
      ./_demo/c/cabisret
      ./_demo/go/cabi
      ./_demo/go/reflect
      ./_demo/go/stdlib
      ./_demo/go/sync
    )
  else
    search_dirs=(./_demo/go/* ./_demo/py/* ./_demo/c/*)
    for d in "${search_dirs[@]}"; do
      if [ -d "$d" ] && [ -n "$(ls "$d"/*.go 2>/dev/null)" ]; then
        cases+=("$d")
      fi
    done
  fi

  for d in "${cases[@]}"; do
    if [ "${RUNNER_OS:-}" = "Windows" ] && [ "$d" = "./_demo/c/thread" ]; then
      echo "SKIP $d (c/pthread exposes the native POSIX API and is not applicable to Windows)"
      continue
    fi
    if [ "${RUNNER_OS:-}" = "Windows" ] && [ "$d" = "./_demo/py/tensor" ] && [ "${LLGO_WINDOWS_NO_TORCH:-0}" = "1" ]; then
      echo "SKIP $d (PyTorch does not publish Windows ARM64 or win32 wheels)"
      continue
    fi
    if [ "$d" = "./_demo/c/llama2-c" ] && [ "${LLGO_RUN_MODEL_DEMOS:-0}" != "1" ]; then
      echo "SKIP $d (model demo runs in scheduled Model Demo workflow)"
      continue
    fi
    if [ "$d" = "./_demo/c/helloc" ]; then
      echo "SKIP $d (WASI C-FFI demo runs in the cross-compile workflow)"
      continue
    fi
    add_case "$d" "" "$d"
  done
fi

total="${#run_dirs[@]}"
failed=0
failed_cases=""

run_case() {
  local dir="$1"
  local target="$2"
  local output_dir="$3"
  local output executable stage_pid=""
  local case_logs="$log_root/$(basename "$output_dir")"
  mkdir -p "$case_logs"
  trap 'if [ -n "${stage_pid:-}" ]; then kill "$stage_pid" 2>/dev/null || true; wait "$stage_pid" 2>/dev/null || true; fi' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
  local -a cmd
  if [ -n "$target" ]; then
    echo "Testing $dir (target=$target)"
  else
    echo "Testing $dir"
  fi
  # The supervisor must be this case shell's direct child. An extra (cd && ...)
  # shell can exit on cancellation while leaving the Python owner alive.
  cd "$dir" || return 1

  if [ -n "$target" ]; then
    if [ "$mode" = "embedded-build-only" ]; then
      cmd=(llgo build -o "$output_dir/firmware.elf")
    else
      cmd=(llgo run)
    fi
    cmd+=("${llgo_run_flags[@]}")
    cmd+=("-target=$target")
    if [ "$emulator" -eq 1 ]; then
      cmd+=("-emulator")
    fi
    cmd+=(".")
    local stage="embedded-build-and-run"
    if [ "$mode" = "embedded-build-only" ]; then
      stage="build"
    fi
    GOTOOLCHAIN=local GOWORK=off "$python" "$repo_root/dev/run_logged_command.py" \
      --timeout "$embedded_timeout" --label "$stage $dir (target=$target)" \
      "$case_logs/$stage.log" "${cmd[@]}" &
    stage_pid=$!
    if wait "$stage_pid"; then
      stage_pid=""
      echo "PASS"
      return
    fi
    stage_pid=""
    echo "FAIL"
    return 1
  fi

  mkdir -p "$output_dir"
  output="$output_dir/demo"
  if [ "${RUNNER_OS:-}" = "Windows" ]; then
    # Match cmd/go's Windows convention: an explicit -o is exact, so request
    # the executable suffix when the artifact must be launched by os/exec.
    output="$output.exe"
  fi
  cmd=(llgo build)
  cmd+=("${llgo_run_flags[@]}")
  cmd+=("-p=1" "-o" "$output" ".")
  GOTOOLCHAIN=local GOWORK=off "$python" "$repo_root/dev/run_logged_command.py" \
    --timeout "$build_timeout" --label "build $dir" "$case_logs/build.log" "${cmd[@]}" &
  stage_pid=$!
  if ! wait "$stage_pid"; then
    stage_pid=""
    echo "FAIL"
    return 1
  fi
  stage_pid=""
  executable="$output"
  if [ ! -f "$executable" ] && [ -f "$output.exe" ]; then
    executable="$output.exe"
  fi
  "$python" "$repo_root/dev/run_logged_command.py" \
    --timeout "$run_timeout" --label "run $dir" "$case_logs/run.log" "$executable" &
  stage_pid=$!
  if wait "$stage_pid"; then
    stage_pid=""
    echo "PASS"
    return
  fi
  stage_pid=""
  echo "FAIL"
  return 1
}

active_pids=()
active_dirs=()
idx=0

wait_for_one() {
  finished_pid=""
  if [ "${BASH_VERSINFO[0]}" -gt 5 ] || { [ "${BASH_VERSINFO[0]}" -eq 5 ] && [ "${BASH_VERSINFO[1]}" -ge 1 ]; }; then
    if wait -n -p finished_pid; then
      finished_status=0
    else
      finished_status=$?
    fi
  fi
  if [ -z "${finished_pid:-}" ]; then
    # wait -n can omit an already completed job. Reap a
    # known PID to preserve its exit status and always drain the queue.
    # This also supports Bash 3.2, which has no wait -n.
    finished_pid="${active_pids[0]}"
    if wait "$finished_pid"; then
      finished_status=0
    else
      finished_status=$?
    fi
  fi
  for active_index in "${!active_pids[@]}"; do
    if [ "${active_pids[$active_index]}" = "$finished_pid" ]; then
      echo "Finished ${active_dirs[$active_index]} (exit $finished_status)"
      if [ "$finished_status" -ne 0 ]; then
        failed=$((failed+1))
        failed_cases="$failed_cases\n* :x: ${active_dirs[$active_index]}"
      fi
      unset 'active_pids[active_index]' 'active_dirs[active_index]'
      active_pids=("${active_pids[@]}")
      active_dirs=("${active_dirs[@]}")
      break
    fi
  done
}

for i in "${!run_dirs[@]}"; do
  d="${run_dirs[$i]}"
  target="${run_targets[$i]}"
  label="${run_labels[$i]}"
  idx=$((idx+1))
  output_dir="$tmp_root/$(printf '%04d' "$idx")"
  echo "Starting $label"
  (run_case "$d" "$target" "$output_dir") &
  pid=$!
  active_pids+=("$pid")
  active_dirs+=("$label")

  while [ "${#active_pids[@]}" -ge "$jobs" ]; do
    wait_for_one
  done
done

while [ "${#active_pids[@]}" -gt 0 ]; do
  wait_for_one
done

echo "=== Done"
echo "$((total-failed))/$total tests passed"

if [ "$failed" -ne 0 ]; then
  echo ":bangbang: Failed demo cases:" | tee -a result.md
  echo -e "$failed_cases" | tee -a result.md
  exit 1
else
  echo ":white_check_mark: All demo tests passed" | tee -a result.md
fi
