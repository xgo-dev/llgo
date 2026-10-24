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

tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

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
  local output executable
  local -a cmd
  if [ -n "$target" ]; then
    echo "Testing $dir (target=$target)"
  else
    echo "Testing $dir"
  fi

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
    if (cd "$dir" && GOTOOLCHAIN=local GOWORK=off "${cmd[@]}"); then
      echo "PASS"
      return
    fi
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
  if ! (cd "$dir" && GOTOOLCHAIN=local GOWORK=off "${cmd[@]}"); then
    echo "FAIL"
    return 1
  fi
  executable="$output"
  if [ ! -f "$executable" ] && [ -f "$output.exe" ]; then
    executable="$output.exe"
  fi
  if (cd "$dir" && "$executable"); then
    echo "PASS"
    return
  fi
  echo "FAIL"
  return 1
}

if [ "$jobs" -le 1 ] || [ "$total" -le 1 ]; then
  idx=0
  for i in "${!run_dirs[@]}"; do
    d="${run_dirs[$i]}"
    target="${run_targets[$i]}"
    label="${run_labels[$i]}"
    idx=$((idx+1))
    output_dir="$tmp_root/$(printf '%04d' "$idx")"
    if ! run_case "$d" "$target" "$output_dir"; then
      failed=$((failed+1))
      failed_cases="$failed_cases\n* :x: $label"
    fi
  done
else
  active_pids=()
  active_dirs=()
  active_logs=()
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
        cat "${active_logs[$active_index]}"
        echo "Finished ${active_dirs[$active_index]} (exit $finished_status)"
        if [ "$finished_status" -ne 0 ]; then
          failed=$((failed+1))
          failed_cases="$failed_cases\n* :x: ${active_dirs[$active_index]}"
        fi
        unset 'active_pids[active_index]' 'active_dirs[active_index]' 'active_logs[active_index]'
        active_pids=("${active_pids[@]}")
        active_dirs=("${active_dirs[@]}")
        active_logs=("${active_logs[@]}")
        break
      fi
    done
  }

  for i in "${!run_dirs[@]}"; do
    d="${run_dirs[$i]}"
    target="${run_targets[$i]}"
    label="${run_labels[$i]}"
    idx=$((idx+1))
    log="$tmp_root/$(printf '%04d' "$idx").log"
    output_dir="$tmp_root/$(printf '%04d' "$idx")"
    echo "Starting $label"
    (run_case "$d" "$target" "$output_dir") >"$log" 2>&1 &
    pid=$!
    active_pids+=("$pid")
    active_dirs+=("$label")
    active_logs+=("$log")

    while [ "${#active_pids[@]}" -ge "$jobs" ]; do
      wait_for_one
    done
  done

  while [ "${#active_pids[@]}" -gt 0 ]; do
    wait_for_one
  done
fi

echo "=== Done"
echo "$((total-failed))/$total tests passed"

if [ "$failed" -ne 0 ]; then
  echo ":bangbang: Failed demo cases:" | tee -a result.md
  echo -e "$failed_cases" | tee -a result.md
  exit 1
else
  echo ":white_check_mark: All demo tests passed" | tee -a result.md
fi
