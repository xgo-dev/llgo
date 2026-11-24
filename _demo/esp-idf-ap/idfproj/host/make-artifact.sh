#!/usr/bin/env bash
# Host-side pipeline: Go program -> IDF-linkable relocatable object.
#
#   1. llgo compiles the Go program into a relocatable object
#      (targets/esp32s31-idf: ld.lld -r, no libc, no linker script).
#   2. The llgo runtime symbol providers (GC heap window, bootstrap stack
#      window - llgo_syms.S) are assembled and merged into the same
#      relocatable. They live inside llgo_out.o rather than the IDF component
#      because GNU ld scans each archive once in link order: IDF repeats the
#      component archives in rounds before llgo_out.o is processed, so a
#      provider inside libmain.a would never be pulled to satisfy llgo_out.o's
#      references. Definitions inside the always-linked object have no such
#      dependency.
#   3. Compiler-rt-style builtins carried by the object (__adddf3, __atomic_*,
#      ...) are renamed with an llgort prefix. They are soft-float/ilp32 ABI
#      versions; IDF's own code resolves its identically-named builtins from
#      its ilp32f libgcc and esp_libc stdatomic - renaming isolates the two
#      ABI worlds while keeping every internal reference consistent.
#   4. The object is presented as single-float ABI, because llgo's RV32 codegen
#      is soft-float ilp32 while IDF compiles esp32s31 with the FPU (ilp32f).
#
# Usage: ./make-artifact.sh [path-to-llgo] (default /tmp/llgo-x)
#
# Also syncs the pkg/esp/*/glue C templates this probe needs into main/
# (integration way 1 in pkg/esp/README.md): wifi_glue.c for the soft AP and
# net_glue.c for the TCP listener. They are gitignored build inputs; this
# script must run before `idf.py build` so CMake configure sees them.
set -euo pipefail

LLGO_BIN="${1:-/tmp/llgo-x}"
DIR="$(cd "$(dirname "$0")" && pwd)"
MAIN="$DIR/../main"
OUT="$MAIN/llgo_out.o"

NM=nm
OBJCOPY=objcopy
CLANG=clang
LLD=ld.lld
# Prefer the esp-clang snapshot in the llgo toolchain cache (same compiler
# family the llgo build used); both cache locations are probed, newest
# snapshot wins. Without it, fall back to plain clang/llvm tools, but only
# if that clang can actually target riscv32 - otherwise fail loudly here
# instead of with a cryptic error three steps later.
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
    echo "       install the esp-clang toolchain (llgo's cache:" >&2
    echo "       ~/Library/Caches/llgo or ~/.cache/llgo, crosscompile/esp-clang-*)" >&2
    echo "       or point CLANG/LLD/NM/OBJCOPY at a working llvm toolchain" >&2
    exit 1
fi

cd "$DIR/../.."

# 0. Sync the two package glues this probe links against into the main
#    component. REPO_ROOT is the llgo checkout this demo lives in, derived
#    from this script's location (host -> idfproj -> esp-idf-ap -> _demo ->
#    repo root); override it via the environment when this project tree lives
#    outside the checkout.
REPO_ROOT="${REPO_ROOT:-$DIR/../../../..}"
for g in wifi net; do
    cp "$REPO_ROOT/pkg/esp/$g/glue/${g}_glue.c" "$MAIN/${g}_glue.c"
done

# 1. Go -> relocatable object.
"$LLGO_BIN" build -target esp32s31-idf -o "$MAIN/llgo_tmp" ./goapp
mv "$MAIN/llgo_tmp.elf" "$MAIN/llgo_out.raw.o"

# 2. Assemble the runtime symbol providers and merge them in. The static GC
#    window defaults to 64 KiB; the collector grows into PSRAM from there
#    (tinygogc/gc_segment_idf.go), so the window only covers hot traffic on a
#    PSRAM board. Raise LLGO_GC_WINDOW_KB for a PSRAM-less board.
GC_WIN_KB="${LLGO_GC_WINDOW_KB:-64}"
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

# 4. Present the object as single-float ABI. llgo's RV32 codegen is
#    soft-float ilp32, while IDF compiles esp32s31 with the FPU (ilp32f) and
#    GNU ld hard-fails the EF_RISCV_FLOAT_ABI mismatch at the final link.
#    Marking the object as float-abi=single is safe for this boundary: every
#    Go<->C signature crossing the link (main, vTaskDelay, xTaskGetTickCount)
#    passes integers only, Go's own soft-float helpers were renamed in step 3,
#    and IDF's float paths stay on its own ilp32f libgcc.
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
# Link-checklist echo: the probe must reference the soft-AP glue symbols; a
# missing llgo_wifi_* / llgo_net_* means the Go program (or a synced glue)
# drifted from the package set this script wires into the IDF build.
echo "llgo_wifi_* glue symbols (must all be in the list above):"
"$NM" -u "$OUT" | grep 'llgo_wifi_' || echo "NONE - goapp does not import pkg/esp/wifi?"
echo "llgo_net_* glue symbols (must all be in the list above):"
"$NM" -u "$OUT" | grep 'llgo_net_' || echo "NONE - goapp does not import pkg/esp/net?"
