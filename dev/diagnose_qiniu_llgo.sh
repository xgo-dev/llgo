#!/usr/bin/env bash
set -euo pipefail

invocation=$(mktemp -d "$DIAGNOSTIC_OUTPUT/invocation.XXXXXX")
printf '%q ' "$@" >"$invocation/command.txt"
printf 'pid=%s started=%s\n' "$$" "$(date -u +%FT%TZ)" >"$invocation/start.txt"
# exec preserves the PID and the original command's arguments and exit status.
exec "$LLGO_REAL" "$@"
