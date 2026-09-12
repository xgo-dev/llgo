#!/usr/bin/env bash
set -euo pipefail

goroot=$(go env GOROOT)
for minute in {1..24}; do
  sleep 60
  if ! kill -0 "$DIAGNOSTIC_PIPELINE" 2>/dev/null; then exit 0; fi
  {
    date -u +%FT%TZ
    ps -eo pid,ppid,pgid,pcpu,pmem,stat,wchan:32,etime,args --forest
    free -h
    for pid in $(pgrep -g "$DIAGNOSTIC_PIPELINE" || true); do
      printf '\nProcess %s\n' "$pid"
      ps -L -p "$pid" -o pid,tid,pcpu,stat,wchan:32,comm || true
      ls -l "/proc/$pid/fd" 2>/dev/null || true
    done
    if [[ -r /sys/fs/cgroup/memory.events ]]; then cat /sys/fs/cgroup/memory.events; fi
  } >"$DIAGNOSTIC_OUTPUT/processes-minute-$minute.txt"
  completed=$(awk '$1 == "ok" && $2 ~ /^github.com\/xgo-dev\/llgo\/test/ {n++} END {print n+0}' "$DIAGNOSTIC_OUTPUT/pipeline.log")
  printf '[diagnostic] minute=%s completed-package-lines=%s; process snapshot saved\n' "$minute" "$completed"
  if [[ "$minute" != 22 && "$minute" != 24 ]]; then continue; fi

  # ptrace can change scheduling, so sample only after the ordinary pipeline
  # should have finished and label any later success as a perturbed trial.
  date -u +%FT%TZ >>"$DIAGNOSTIC_OUTPUT/ptrace-sampled.txt"
  sampled=0
  for pid in $(pgrep -g "$DIAGNOSTIC_PIPELINE" || true); do
    executable=$(readlink "/proc/$pid/exe" || true)
    case "$executable" in
      */llgo|*/runner-*|*.test)
        # The compiler is a Go executable; native tests need native stacks.
        # Go's own gdb helpers also expose parked compiler goroutines.
        # shellcheck disable=SC2024
        sudo timeout -k 2s 12s gdb --batch --nx -p "$pid" \
          -ex 'set pagination off' \
          -ex 'thread apply all bt 30' \
          -ex "source $goroot/src/runtime/runtime-gdb.py" \
          -ex 'goroutine all bt 20' -ex detach \
          >"$DIAGNOSTIC_OUTPUT/stacks-minute-$minute-pid-$pid.txt" 2>&1 || true
        sampled=$((sampled + 1))
        if (( sampled >= 6 )); then break; fi
        ;;
    esac
  done
done
