#!/usr/bin/env bash
# Acceptance run for the llgo ESP-NOW probe under esp-emu.
#
# Prerequisites: idfproj/build/merged-binary.bin (host/make-artifact.sh +
# `idf.py build && idf.py merge-bin`, see the README), esp-emu v1.2.0.
#
# What it proves end to end: the whole pkg/esp/espnow control plane runs
# through the binding inside the emulator - bring-up in the documented
# order (WiFi first), the peer-list lifecycle with the driver's own error
# codes, the synchronous send refusals, broadcast sends with their
# asynchronous SendResult confirmations resolved through the pump, the
# encryption entry points, and a clean Deinit/re-Init cycle. No frame is
# ever RECEIVED: the emulator has no second ESP-NOW device (two esp-emu
# instances do not hear each other - measured, see the README), so the
# receive path stays a hardware item and the run says so honestly.
#
# The run is serialized against every other esp-emu use on this machine
# (/tmp/esp-emu.lock): the lock wait is bounded, a stale lock is reported
# with its age instead of hanging the script, and the lock is released on
# every exit path. The firmware's console lines are compared against the
# golden (golden/emu-espnow-probe.txt) after normalizing the one value that
# legitimately varies between runs (the emulator's synthetic station MAC);
# --no-golden skips that and only checks the PASS markers.
#
# Usage: ./run-acceptance.sh [--no-golden]
#   FW / EMU / LOG / TIMEOUT / LOCK_TIMEOUT / LOCK_STALE_SECS override the
#   defaults below.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
DEMO="$(cd "$DIR/../.." && pwd)"
FW="${FW:-$DEMO/idfproj/build/merged-binary.bin}"
EMU="${EMU:-$HOME/.local/bin/esp-emu}"
LOG="${LOG:-$DIR/logs/espnow-acceptance.log}"
TIMEOUT="${TIMEOUT:-90s}"
LOCK=/tmp/esp-emu.lock
LOCK_TIMEOUT="${LOCK_TIMEOUT:-900}"     # seconds to wait for the lock
LOCK_STALE_SECS="${LOCK_STALE_SECS:-1800}"  # older than this: report and bail
GOLDEN_CHECK=1
[[ "${1:-}" == "--no-golden" ]] && GOLDEN_CHECK=0

if [[ ! -f "$FW" ]]; then
    echo "error: no firmware at $FW" >&2
    echo "       run idfproj/host/make-artifact.sh and then" >&2
    echo "       'idf.py build && idf.py merge-bin -o merged-binary.bin' first" >&2
    exit 1
fi
if [[ ! -x "$EMU" ]]; then
    echo "error: no emulator at $EMU (set EMU=/path/to/esp-emu)" >&2
    exit 1
fi

mkdir -p "$(dirname "$LOG")"
: > "$LOG"

# The emulator is heavy (a full S31 + PSRAM); one instance at a time, and no
# unbounded wait for the lock.
waited=0
while ! mkdir "$LOCK" 2>/dev/null; do
    if (( waited == 0 )); then
        echo "waiting for $LOCK (another esp-emu run is in progress)..."
    fi
    if (( waited >= LOCK_TIMEOUT )); then
        age="unknown"
        if stat -f %m "$LOCK" >/dev/null 2>&1; then
            age="$(( $(date +%s) - $(stat -f %m "$LOCK") ))s"
        elif stat -c %Y "$LOCK" >/dev/null 2>&1; then
            age="$(( $(date +%s) - $(stat -c %Y "$LOCK") ))s"
        fi
        echo "error: $LOCK still held after ${LOCK_TIMEOUT}s (age $age)" >&2
        if [[ "$age" != "unknown" ]] && (( ${age%s} > LOCK_STALE_SECS )); then
            echo "       the lock looks stale; if no esp-emu is running, rmdir $LOCK" >&2
        fi
        exit 1
    fi
    sleep 5
    waited=$(( waited + 5 ))
done

EMU_PID=""
release() {
    [[ -n "$EMU_PID" ]] && kill "$EMU_PID" 2>/dev/null || true
    rmdir "$LOCK" 2>/dev/null || true
}
trap release EXIT
trap 'release; exit 130' INT
trap 'release; exit 143' TERM

echo "firmware: $FW"
echo "log:      $LOG"
"$EMU" --chip esp32s31 --firmware "$FW" \
    --console uart0 --psram-size 16M \
    --exit-on "[go] espnow done" --timeout "$TIMEOUT" \
    >"$LOG" 2>&1 || true
EMU_PID=""

STATUS=0
if grep -q '^\[go\] espnow PASS' "$LOG" && grep -q '^\[go\] espnow done' "$LOG"; then
    echo "firmware reached espnow PASS and espnow done"
else
    echo "ACCEPTANCE FAIL: the firmware did not reach espnow PASS/done (log: $LOG)" >&2
    grep -E '^\[go\] espnow FAIL' "$LOG" >&2 || true
    STATUS=1
fi

# Golden comparison. One value legitimately varies between runs and is
# normalized before the diff: the emulator's synthetic station MAC.
if (( GOLDEN_CHECK )); then
    normalize() {
        tr -d '\r' | sed -E \
            -e 's/(own sta mac = )[0-9a-f:]+/\1MAC/'
    }
    ACTUAL=$(mktemp)
    EXPECT=$(mktemp)
    trap 'rm -f "$ACTUAL" "$EXPECT"; release' EXIT INT TERM
    grep -E '^\[go\] espnow' "$LOG" | normalize > "$ACTUAL"
    grep -v '^#' "$DIR/golden/emu-espnow-probe.txt" | normalize > "$EXPECT"
    if diff -u "$EXPECT" "$ACTUAL"; then
        echo "golden: MATCH ($(wc -l < "$EXPECT" | tr -d ' ') lines)"
    else
        echo "ACCEPTANCE FAIL: firmware lines differ from golden/emu-espnow-probe.txt" >&2
        STATUS=1
    fi
fi

if (( STATUS == 0 )); then
    echo "ACCEPTANCE PASS (log: $LOG)"
else
    echo "ACCEPTANCE FAIL (log: $LOG)" >&2
fi
exit "$STATUS"
