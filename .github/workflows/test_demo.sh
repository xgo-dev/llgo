#!/bin/bash
set -e

mode="host"
if [ "${1:-}" = "--embedded" ]; then
  mode="embedded"
  shift
fi

# llgo run subdirectories under _demo that contain *.go files
jobs="${LLGO_DEMO_JOBS:-1}"
if [ "${jobs}" -gt 1 ]; then
  if [ "${BASH_VERSINFO[0]}" -lt 5 ] || { [ "${BASH_VERSINFO[0]}" -eq 5 ] && [ "${BASH_VERSINFO[1]}" -lt 1 ]; }; then
    echo "warning: LLGO_DEMO_JOBS=${jobs} requested but bash ${BASH_VERSION} lacks 'wait -n -p'; running sequentially" >&2
    jobs=1
  fi
fi
tmp_root="$(mktemp -d)"
trap 'rm -rf "$tmp_root"' EXIT

cases=()
if [ "$mode" = "embedded" ]; then
  while IFS= read -r dir; do
    cases+=("$dir")
  done < <(find ./_demo/go ./_demo/c -name '*.go' -print | xargs -n1 dirname | sort -u)
else
  search_dirs=(./_demo/go/* ./_demo/py/* ./_demo/c/*)
  for d in "${search_dirs[@]}"; do
    if [ -d "$d" ] && [ -n "$(ls "$d"/*.go 2>/dev/null)" ]; then
      cases+=("$d")
    fi
  done
fi

embedded_targets=()
emulator=0
if [ "$mode" = "embedded" ]; then
  emulator=1
  embedded_targets=(esp32 esp32c3-basic)
fi

ignore_esp32=(
  "./_demo/c/asmcall" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/asmfullcall" # panic: cannot build SSA for packages
  "./_demo/c/cabisret" # timeout: emulator did not auto-exit
  "./_demo/c/cargs" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/cexec" # link error: ld.lld undefined symbol execlp
  "./_demo/c/cgofull" # fast fail: build constraints exclude all Go files
  "./_demo/c/cgofull/pymod1" # fast fail: build constraints exclude all Go files
  "./_demo/c/cgofull/pymod2" # fast fail: build constraints exclude all Go files
  "./_demo/c/concat" # link error: ld.lld undefined symbol stderr
  "./_demo/c/cppintf" # C++ compile error: libc++ reports "No thread API"
  "./_demo/c/cppintf/foo" # C++ compile error: libc++ reports "No thread API"
  "./_demo/c/cppmintf" # C++ compile error: libc++ reports "No thread API"
  "./_demo/c/cppmintf/foo" # C++ compile error: libc++ reports "No thread API"
  "./_demo/c/cppstr" # C++ compile error: libc++ reports "No thread API"
  "./_demo/c/crand" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/ctime" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/getcwd" # timeout: emulator did not auto-exit
  "./_demo/c/hello" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/llama2-c" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/netdbdemo" # link error: ld.lld undefined symbol getaddrinfo
  "./_demo/c/setjmp" # panic: cannot build SSA for packages
  "./_demo/c/socket/client" # link error: ld.lld undefined symbol socket
  "./_demo/c/socket/server" # link error: ld.lld undefined symbol socket
  "./_demo/c/stacksave" # fast fail: build constraints exclude all Go files
  "./_demo/c/syncdebug" # fast fail: build constraints exclude all Go files in pthread/sync
  "./_demo/c/thread" # link error: ld.lld undefined symbol GC_pthread_create
  "./_demo/go/abimethod" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/async" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/async/timeout" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/cabi" # runtime output: fatal error
  "./_demo/go/cgo" # fast fail: build constraints exclude all Go files
  "./_demo/go/checkfile" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/commandrun" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/createtemp-1654" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/defer" # runtime output: fatal error
  "./_demo/go/embedunexport-1598" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/export" # timeout: emulator did not auto-exit
  "./_demo/go/failed/stacktrace" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gobuild" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gobuild-1389" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/goimporter-1389" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/goroutine" # timeout: emulator did not auto-exit
  "./_demo/go/gotime" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gotoken" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gotypes" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/issue1538" # timeout: emulator did not auto-exit
  "./_demo/go/issue1538-floatcvtuint-over" # timeout: emulator did not auto-exit
  "./_demo/go/logdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/maphash" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/mimeheader" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/mkdirdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/netip" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/osfile" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/oslookpath" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/oswritestring" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/randcrypt" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/randdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/readdir" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectcopy" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectfunc" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectindirect" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectmake" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectmethod" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectname-1412" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectpointerto" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/return-1605" # runtime output: fatal error
  "./_demo/go/runtime" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sync" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/syscall" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sysexec" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sysopen-1654" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/texttemplate" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/timedur" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/timer" # panic: internal/bytealg selected .s files require plan9asm translation
)

ignore_esp32c3_basic=(
  "./_demo/go/mkdirdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/asmcall" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/asmfullcall" # panic: cannot build SSA for packages (undefined: verify)
  "./_demo/c/cargs" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/catomic" # link error: ld.lld undefined symbol __atomic_store
  "./_demo/c/cexec" # link error: ld.lld undefined symbol execlp
  "./_demo/c/cgofull" # fast fail: build constraints exclude all Go files
  "./_demo/c/cgofull/pymod1" # fast fail: build constraints exclude all Go files
  "./_demo/c/cgofull/pymod2" # fast fail: build constraints exclude all Go files
  "./_demo/c/concat" # link error: ld.lld undefined symbol stderr
  "./_demo/c/cppstr" # C++ compile error: '<string>' file not found
  "./_demo/c/crand" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/ctime" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/getcwd" # timeout: emulator did not auto-exit
  "./_demo/c/hello" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/c/llama2-c" # fast fail: build constraints exclude all Go files in lib/c/time
  "./_demo/c/netdbdemo" # link error: ld.lld undefined symbol getaddrinfo
  "./_demo/c/setjmp" # panic: cannot build SSA for packages (undefined SigjmpBuf/Siglongjmp)
  "./_demo/c/socket/client" # link error: ld.lld undefined symbol socket
  "./_demo/c/socket/server" # link error: ld.lld undefined symbol socket
  "./_demo/c/stacksave" # fast fail: build constraints exclude all Go files
  "./_demo/c/syncdebug" # fast fail: build constraints exclude all Go files in pthread/sync
  "./_demo/c/thread" # link error: ld.lld undefined symbol GC_pthread_create
  "./_demo/go/abimethod" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/async" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/async/timeout" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/checkfile" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/commandrun" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/createtemp-1654" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/cgo" # fast fail: build constraints exclude all Go files
  "./_demo/go/embedunexport-1598" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/export" # link error: ld.lld undefined symbol __atomic_fetch_or_4
  "./_demo/go/failed/stacktrace" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gobuild" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gobuild-1389" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/goimporter-1389" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/goroutine" # timeout: emulator did not auto-exit
  "./_demo/go/gotime" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gotoken" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/gotypes" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/logdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/maphash" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/mimeheader" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/netip" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/osfile" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/oslookpath" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/oswritestring" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/randcrypt" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/randdemo" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/readdir" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectfunc" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectindirect" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectcopy" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectmethod" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectmake" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectname-1412" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/reflectpointerto" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/runtime" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sync" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sysopen-1654" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/syscall" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/sysexec" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/texttemplate" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/timedur" # panic: internal/bytealg selected .s files require plan9asm translation
  "./_demo/go/timer" # panic: internal/bytealg selected .s files require plan9asm translation
)

should_ignore() {
  local dir="$1"
  local target="$2"
  case "$target" in
    esp32)
      for ignore in "${ignore_esp32[@]}"; do
        if [ "$dir" = "$ignore" ]; then
          return 0
        fi
      done
      ;;
    esp32c3-basic)
      for ignore in "${ignore_esp32c3_basic[@]}"; do
        if [ "$dir" = "$ignore" ]; then
          return 0
        fi
      done
      ;;
  esac
  return 1
}

run_dirs=()
run_targets=()
run_labels=()

if [ "$mode" = "embedded" ]; then
  for target in "${embedded_targets[@]}"; do
    for d in "${cases[@]}"; do
      if should_ignore "$d" "$target"; then
        echo "SKIP $d (target=$target)"
        continue
      fi
      run_dirs+=("$d")
      run_targets+=("$target")
      run_labels+=("$d (target=$target)")
    done
  done
else
  for d in "${cases[@]}"; do
    run_dirs+=("$d")
    run_targets+=("")
    run_labels+=("$d")
  done
fi

total="${#run_dirs[@]}"
failed=0
failed_cases=""

run_case() {
  local dir="$1"
  local target="$2"
  if [ -n "$target" ]; then
    echo "Testing $dir (target=$target)"
  else
    echo "Testing $dir"
  fi
  cmd=(llgo run)
  if [ -n "$target" ]; then
    cmd+=("-target=$target")
  fi
  if [ "$emulator" -eq 1 ]; then
    cmd+=("-emulator")
  fi
  cmd+=(".")
  if (cd "$dir" && GOWORK=off "${cmd[@]}"); then
    echo "PASS"
  else
    echo "FAIL"
    return 1
  fi
}

if [ "$jobs" -le 1 ] || [ "$total" -le 1 ]; then
  for i in "${!run_dirs[@]}"; do
    d="${run_dirs[$i]}"
    target="${run_targets[$i]}"
    label="${run_labels[$i]}"
    if ! run_case "$d" "$target"; then
      failed=$((failed+1))
      failed_cases="$failed_cases\n* :x: $label"
    fi
  done
else
  active_pids=()
  active_dirs=()
  active_logs=()
  idx=0

  for i in "${!run_dirs[@]}"; do
    d="${run_dirs[$i]}"
    target="${run_targets[$i]}"
    label="${run_labels[$i]}"
    idx=$((idx+1))
    log="$tmp_root/$(printf '%04d' "$idx").log"
    (run_case "$d" "$target") >"$log" 2>&1 &
    pid=$!
    active_pids+=("$pid")
    active_dirs+=("$label")
    active_logs+=("$log")

    while [ "${#active_pids[@]}" -ge "$jobs" ]; do
      finished_pid=""
      if wait -n -p finished_pid; then
        finished_status=0
      else
        finished_status=$?
      fi
      for i in "${!active_pids[@]}"; do
        if [ "${active_pids[$i]}" = "$finished_pid" ]; then
          cat "${active_logs[$i]}"
          if [ "$finished_status" -ne 0 ]; then
            failed=$((failed+1))
            failed_cases="$failed_cases\n* :x: ${active_dirs[$i]}"
          fi
          unset 'active_pids[i]' 'active_dirs[i]' 'active_logs[i]'
          active_pids=("${active_pids[@]}")
          active_dirs=("${active_dirs[@]}")
          active_logs=("${active_logs[@]}")
          break
        fi
      done
    done
  done

  while [ "${#active_pids[@]}" -gt 0 ]; do
    finished_pid=""
    if wait -n -p finished_pid; then
      finished_status=0
    else
      finished_status=$?
    fi
    for i in "${!active_pids[@]}"; do
      if [ "${active_pids[$i]}" = "$finished_pid" ]; then
        cat "${active_logs[$i]}"
          if [ "$finished_status" -ne 0 ]; then
            failed=$((failed+1))
            failed_cases="$failed_cases\n* :x: ${active_dirs[$i]}"
          fi
        unset 'active_pids[i]' 'active_dirs[i]' 'active_logs[i]'
        active_pids=("${active_pids[@]}")
        active_dirs=("${active_dirs[@]}")
        active_logs=("${active_logs[@]}")
        break
      fi
    done
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
