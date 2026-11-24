#!/usr/bin/env bash
# Acceptance run for the llgo soft-AP probe under esp-emu.
#
# Prerequisites: idfproj/build/merged-binary.bin (host/make-artifact.sh +
# `idf.py build && idf.py merge-bin`, see the README), esp-emu v1.2.0, and a
# Go toolchain for the host client.
#
# What it proves end to end: the emulator joins the firmware's own soft AP as a
# station, the firmware's DHCP server leases it an address, and a plain host TCP
# connection reaches a listener inside the firmware through that link.
#
# The run is serialized against every other esp-emu use on this machine
# (/tmp/esp-emu.lock): the lock wait is bounded, a stale lock is reported with
# its age instead of hanging the script, and the lock is released on every exit
# path. The firmware's console lines are compared against the golden
# (golden/emu-ap-probe.txt) after normalizing the four values that legitimately
# vary between runs; --no-golden skips that and only checks the PASS markers.
#
# Usage: ./run-acceptance.sh [--no-golden]
#   FW / EMU / HOST_PORT / LOG / TIMEOUT / LOCK_TIMEOUT / LOCK_STALE_SECS
#   override the defaults below.
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
DEMO="$(cd "$DIR/../.." && pwd)"
FW="${FW:-$DEMO/idfproj/build/merged-binary.bin}"
EMU="${EMU:-$HOME/.local/bin/esp-emu}"
FW_PORT=8000
HOST_PORT="${HOST_PORT:-18080}"
LOG="${LOG:-$DIR/logs/ap-acceptance.log}"
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

# Host port precheck: a leftover client or a previous run holding it would make
# the hostfwd bind fail inside esp-emu, minutes after the interesting output.
if command -v lsof >/dev/null 2>&1 && lsof -nP -iTCP:"$HOST_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "error: 127.0.0.1:$HOST_PORT is already in use; free it or set HOST_PORT" >&2
    lsof -nP -iTCP:"$HOST_PORT" -sTCP:LISTEN >&2 || true
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
    --net "user,hostfwd=tcp::$HOST_PORT-:$FW_PORT" \
    --wifi-ssid myssid --wifi-password mypassword \
    --wifi-fw-ap-password llgoappass \
    --exit-on "[go] ap done" --timeout "$TIMEOUT" \
    >"$LOG" 2>&1 &
EMU_PID=$!

# Wait for the firmware to report the AP up (its listener is bound at the same
# point), then dial through hostfwd.
for _ in $(seq 1 120); do
    if grep -q '^\[go\] ap ready' "$LOG"; then break; fi
    if ! kill -0 "$EMU_PID" 2>/dev/null; then
        echo "error: the emulator exited before the AP came up; see $LOG" >&2
        exit 1
    fi
    sleep 1
done
if ! grep -q '^\[go\] ap ready' "$LOG"; then
    echo "error: the firmware never reported [go] ap ready; see $LOG" >&2
    tail -20 "$LOG" >&2 || true
    exit 1
fi

CLIENT_STATUS=0
(cd "$DEMO" && go run ./idfproj/host/apclient \
    -addr "127.0.0.1:$HOST_PORT" -request "hello llgo-ap" -want "llgo-ap-ok") \
    || CLIENT_STATUS=$?

wait "$EMU_PID" || true
EMU_PID=""

echo "---- firmware AP lines ----"
grep -E '^\[(go|c)\] (ap|wifi)' "$LOG" || true
echo "---- host client ----"
[[ $CLIENT_STATUS -eq 0 ]] || { echo "host client failed (status $CLIENT_STATUS)"; exit "$CLIENT_STATUS"; }

STATUS=0
if grep -q '^\[go\] ap PASS' "$LOG" && grep -q '^\[go\] ap done' "$LOG"; then
    echo "firmware reached ap PASS and ap done"
else
    echo "ACCEPTANCE FAIL: the firmware did not reach ap PASS/ap done (log: $LOG)" >&2
    STATUS=1
fi

# Golden comparison. Four values legitimately vary between runs and are
# normalized before the diff: how many clients the emulator had already joined
# with when the first list was read, the driver's AID, the emulator's synthetic
# RSSI, and the 802.11 reason code.
if (( GOLDEN_CHECK )); then
    normalize() {
        tr -d '\r' | sed -E \
            -e 's/(stations on start n = )[0-9]+/\1N/' \
            -e 's/(aid = )[0-9]+/\1N/' \
            -e 's/(rssi = )-?[0-9]+/\1N/' \
            -e 's/(reason = )[0-9]+/\1N/'
    }
    ACTUAL=$(mktemp)
    EXPECT=$(mktemp)
    trap 'rm -f "$ACTUAL" "$EXPECT"; release' EXIT INT TERM
    grep -E '^\[go\] (ap|wifi)' "$LOG" | grep -v 'heap ' | normalize > "$ACTUAL"
    grep -v '^#' "$DIR/golden/emu-ap-probe.txt" | normalize > "$EXPECT"
    if diff -u "$EXPECT" "$ACTUAL"; then
        echo "golden: MATCH ($(wc -l < "$EXPECT" | tr -d ' ') lines)"
    else
        echo "ACCEPTANCE FAIL: firmware lines differ from golden/emu-ap-probe.txt" >&2
        STATUS=1
    fi
fi

if (( STATUS == 0 )); then
    echo "ACCEPTANCE PASS (log: $LOG)"
else
    echo "ACCEPTANCE FAIL (log: $LOG)" >&2
fi
exit "$STATUS"
