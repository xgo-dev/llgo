#!/usr/bin/env bash
# Host-side pipeline: Go program -> IDF-linkable relocatable object.
#
#   1. llgo compiles the Go program into a relocatable object
#      (targets/esp32s31-idf: ld.lld -r, no libc, no linker script).
#   2. The llgo runtime symbol providers (GC heap window, bootstrap stack
#      window - llgo_syms.S) are assembled and merged into the same
#      relocatable; see _demo/esp-idf-ap/idfproj/host/make-artifact.sh for
#      the full rationale (archive scan order, builtin renaming, float ABI).
#
# Glue syncing: the pkg/esp glue templates are build inputs synced from the
# llgo repo (integration way 1 in pkg/esp/README.md) and therefore
# gitignored. This firmware links eight of them:
#
#   wifi_glue.c net_glue.c bt_glue.c i2c_glue.c i2s_glue.c gpio_glue.c
#       verbatim copies (bt from pkg/esp/bt/glue/bt_glue_bluedroid.c).
#   a2dp_glue.c     pkg/esp/bt/glue/bt_glue_a2dp.c + the one-line MCLK patch
#                   below (the board's ES8311 needs MCLK on GPIO52; the
#                   stock glue leaves it unused - recorded as a pkg/esp/bt
#                   capability gap: A2DPConfig has no MCLK field).
#   ble_glue.c      NOT synced: a checked-in copy of
#                   pkg/esp/ble/glue/ble_glue_bluedroid.c carrying this
#                   firmware's local patches (shared-stack bring-up next to
#                   the classic host, and the BLE pairing flow the keyboard
#                   verification needs). See its file header for the delta.
#
# Usage: ./make-artifact.sh [path-to-llgo] (default /tmp/llgo-lp2)
set -euo pipefail

LLGO_BIN="${1:-/tmp/llgo-lp2}"
DIR="$(cd "$(dirname "$0")" && pwd)"
MAIN="$DIR/../main"
OUT="$MAIN/llgo_out.o"

# The llgo checkout the pkg/esp glues sync from: this project tree lives
# inside the checkout (llgo/_demo/esp-idf-all/idfproj/host), so the repo
# root is four levels up.
REPO_ROOT="${REPO_ROOT:-$(cd "$DIR/../../../.." && pwd)}"

NM=nm
OBJCOPY=objcopy
CLANG=clang
LLD=ld.lld
for CACHE in "$HOME/Library/Caches/llgo" "$HOME/.cache/llgo"; do
    ESPCLANG=$(ls -d "$CACHE"/crosscompile/esp-clang-*/bin/clang 2>/dev/null | sort | tail -1)
    if [[ -n "$ESPCLANG" && -x "$ESPCLANG" ]]; then
        ESPBIN="$(dirname "$ESPCLANG")"
        CLANG="$ESPBIN/clang"
        LLD="$ESPBIN/ld.lld"
        NM="$ESPBIN/llvm-nm"
        OBJCOPY="$ESPBIN/llvm-objcopy"
        break
    fi
done
if ! echo 'int x;' | "$CLANG" -x c --target=riscv32-esp-elf -march=rv32imc_zicsr -c - -o /dev/null 2>/dev/null; then
    echo "error: $CLANG cannot compile for riscv32-esp-elf" >&2
    exit 1
fi

cd "$DIR/../.."

# 0. Sync the package glues this probe links against into the main component
#    (build inputs, like in the ap demo). ble_glue.c is the checked-in
#    patched copy - see the block comment at the top.
for g in wifi net i2c i2s gpio; do
    cp "$REPO_ROOT/pkg/esp/$g/glue/${g}_glue.c" "$MAIN/${g}_glue.c"
done
cp "$REPO_ROOT/pkg/esp/bt/glue/bt_glue_bluedroid.c" "$MAIN/bt_glue.c"

# Diagnostic (LLGO_BT_TRACE=1): stage prints around the classic-stack
# bring-up, to locate a hang inside one of the blocking C calls. Remove
# once the hang is root-caused.
if [[ "${LLGO_BT_TRACE:-0}" == "1" ]]; then
python3 - "$MAIN/bt_glue.c" <<'PYEOF'
import sys
path = sys.argv[1]
src = open(path).read()
assert '#include <stdio.h>' not in src
src = src.replace('#include "esp_bt.h"', '#include <stdio.h>\n#include "esp_bt.h"', 1)
marks = [
    ('    if (s_uninit_sem == NULL) {',
     '    printf("[c] bt trace: nvs ok\\n");\n    if (s_uninit_sem == NULL) {'),
    ('    err = esp_bt_controller_init(&cfg);',
     '    printf("[c] bt trace: ctrl init...\\n");\n    err = esp_bt_controller_init(&cfg);\n    printf("[c] bt trace: ctrl init rc=%d\\n", (int)err);'),
    ('    err = esp_bt_controller_enable(ESP_BT_MODE_BTDM);',
     '    printf("[c] bt trace: ctrl enable BTDM...\\n");\n    err = esp_bt_controller_enable(ESP_BT_MODE_BTDM);\n    printf("[c] bt trace: ctrl enable rc=%d\\n", (int)err);'),
    ('    err = esp_bluedroid_init();',
     '    printf("[c] bt trace: bluedroid init...\\n");\n    err = esp_bluedroid_init();\n    printf("[c] bt trace: bluedroid init rc=%d\\n", (int)err);'),
    ('    err = esp_bluedroid_enable();',
     '    printf("[c] bt trace: bluedroid enable...\\n");\n    err = esp_bluedroid_enable();\n    printf("[c] bt trace: bluedroid enable rc=%d\\n", (int)err);'),
    ('    err = esp_bt_gap_register_callback(gap_cb);',
     '    printf("[c] bt trace: gap register...\\n");\n    err = esp_bt_gap_register_callback(gap_cb);\n    printf("[c] bt trace: gap register rc=%d ready=1\\n", (int)err);'),
]
for old, new in marks:
    assert src.count(old) == 1, "bt trace patch point not found: " + old
    src = src.replace(old, new, 1)
open(path, "w").write(src)
print("bt_glue.c: LLGO_BT_TRACE stage prints applied")
PYEOF
fi

# a2dp_glue.c: the repo template plus the MCLK patch. The assert keeps a
# drifted template from silently building without MCLK - the codec would
# answer every register write and stay silent forever.
cp "$REPO_ROOT/pkg/esp/bt/glue/bt_glue_a2dp.c" "$MAIN/a2dp_glue.c"
python3 - "$MAIN/a2dp_glue.c" <<'PYEOF'
import sys
path = sys.argv[1]
src = open(path).read()
old = ".mclk = I2S_GPIO_UNUSED,"
new = (".mclk = (gpio_num_t)52,"
       " /* esp-idf-all local patch: the board's ES8311 needs MCLK;"
       " stock glue leaves it unused */")
assert src.count(old) == 1, "a2dp glue template drifted: MCLK patch point not found"
open(path, "w").write(src.replace(old, new, 1))
print("a2dp_glue.c: MCLK patch applied (GPIO52)")
PYEOF

# 1. Go -> relocatable object (default 2 KiB goroutine fiber stacks; see
#    the controlLoop comment in goapp/main.go for why the deep-C BLE and
#    esp_wifi control calls deliberately run on the main fiber instead of
#    raising every fiber's stack).
"$LLGO_BIN" build -target esp32s31-idf -o "$MAIN/llgo_tmp" ./goapp
mv "$MAIN/llgo_tmp.elf" "$MAIN/llgo_out.raw.o"

# 2. Assemble the runtime symbol providers and merge them in. The static GC
#    window is 40 KiB: big enough that the bootstrap and every goroutine
#    fiber stack (2 KiB each, plus the main fiber's 8 KiB) stay in this
#    INTERNAL region for the whole run - a fiber stack that lands in a PSRAM
#    heap segment dies at the first flash cache-disabled section (NVS) -
#    and big enough for the hot short-lived traffic; the 192 KiB capture
#    buffer and the bulk of the heap grow into PSRAM segments.
GC_WIN_KB="${LLGO_GC_WINDOW_KB:-40}"
"$CLANG" --target=riscv32-esp-elf -march=rv32imc_zicsr \
    -DLLGO_GC_WINDOW_BYTES=$((GC_WIN_KB * 1024)) \
    -c "$DIR/llgo_syms.S" -o "$DIR/llgo_syms.o"
echo "GC window: ${GC_WIN_KB} KiB"
"$LLD" -r -m elf32lriscv "$MAIN/llgo_out.raw.o" "$DIR/llgo_syms.o" -o "$MAIN/llgo_out.merged.o"

# 3. Rename the carried builtins away from libgcc/esp_libc names.
SYMS=$("$NM" -g --defined-only "$MAIN/llgo_out.merged.o" | awk '{print $NF}' \
       | grep -E '^__' | grep -vE '^__(wrap_|llgo_)' | sort -u)
ARGS=()
for s in $SYMS; do
    ARGS+=("--redefine-sym=$s=llgort$s")
done
ARGS+=("--redefine-sym=__malloc_sbrk_start=llgo_picolibc_sbrk_start")
ARGS+=("--redefine-sym=__malloc_sbrk_top=llgo_picolibc_sbrk_top")
"$OBJCOPY" "${ARGS[@]}" "$MAIN/llgo_out.merged.o" "$OUT"

# 4. Present the object as single-float ABI (llgo RV32 codegen is soft-float
#    ilp32; IDF compiles esp32s31 with the FPU).
python3 - "$OUT" <<'PYEOF'
import struct, sys
path = sys.argv[1]
with open(path, 'r+b') as f:
    data = f.read(40)
    e_flags_off = 36
    flags = struct.unpack_from('<I', data, e_flags_off)[0]
    flags = (flags & ~0x6) | 0x2  # EF_RISCV_FLOAT_ABI_SINGLE
    f.seek(e_flags_off)
    f.write(struct.pack('<I', flags))
print('patched EF_RISCV float ABI to single:', path)
PYEOF

rm -f "$MAIN/llgo_out.raw.o" "$MAIN/llgo_out.merged.o"
SIZE=$(stat -f %z "$OUT" 2>/dev/null || stat -c %s "$OUT")
echo "artifact written: $OUT ($SIZE bytes)"
echo "undefined symbols remaining (all must resolve at the IDF link):"
"$NM" -u "$OUT"
echo "glue symbols (must all be in the list above):"
"$NM" -u "$OUT" | grep -E 'llgo_(wifi|net|bt|ble|i2c|i2s|gpio)_' | head -40 || echo "NONE - goapp does not import the pkg/esp packages?"
