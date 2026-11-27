#!/usr/bin/env bash
# Host-side pipeline: Go program -> IDF-linkable relocatable object.
#
# Every demo under _demo/esp-idf-*/idfproj/host/ points its make-artifact.sh at
# this file (a symlink); what differs between demos lives in the sibling
# artifact.conf next to the symlink, and in an optional post-sync.sh hook. The
# demo directory is resolved from the invoking path ($0 keeps the symlink's
# location), so one implementation serves all of them.
#
# What the pipeline does, and why each step exists:
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
#   4. The object is marked single-float ABI: llgo emits soft-float ilp32 while
#      IDF compiles this chip with the FPU (ilp32f), and GNU ld hard-fails the
#      EF_RISCV_FLOAT_ABI mismatch.
#
# Before all that it syncs the pkg/esp/*/glue C templates into main/
# (integration way 1 in pkg/esp/README.md): the Go program imports
# github.com/xgo-dev/llgo/pkg/esp packages, whose llgo.idf externs resolve
# against these glues at the IDF link. They are gitignored build inputs, so
# this script must run before `idf.py build` for CMake configure to see them.
#
# Usage: ./make-artifact.sh [path-to-llgo]   (default $LLGO_BIN or /tmp/llgo-idf)
#
# artifact.conf keys (shell assignments, sourced):
#   GLUES="wifi gpio i2c i2s net tls"   pkg/esp/<g>/glue/<g>_glue.c -> main/<g>_glue.c
#   GLUE_MAP="bt/glue/bt_glue_bluedroid.c:bt_glue.c ..."
#                                       explicit template:destination pairs, for
#                                       templates whose file name differs
#   GC_WIN_KB=64                        static GC window (default 64)
#   GO_TASK_STACK=1                     spawn "llgo-go" on a dedicated stack
#                                       below the GC window (FPU-safe; see
#                                       app_main.c)
#   LLGO_EXTRA_ARGS="..."               extra arguments for the llgo build
#
# post-sync.sh (optional, same directory): sourced after the glue sync, for a
# demo that patches a template it just copied (assert the patch point, so a
# drifted upstream template cannot silently build).

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"   # the demo's host/ directory
MAIN="$DIR/../main"
OUT="$MAIN/llgo_out.o"
COMMON="$(cd "$DIR/../../../common" && pwd)"

NM=${NM:-nm}
OBJCOPY=${OBJCOPY:-objcopy}
CLANG=${CLANG:-clang}
LLD=${LLD:-ld.lld}

CONF="$DIR/artifact.conf"
[[ -f "$CONF" ]] || { echo "missing $CONF (each demo states its glue set there)" >&2; exit 2; }
# shellcheck source=/dev/null
. "$CONF"

# Prefer the esp-clang snapshot in the llgo toolchain cache (the same
# compiler family the llgo build used); both cache locations are probed,
# newest snapshot wins. Without it, fall back to plain clang/llvm tools, but
# only if that clang can actually target riscv32 - otherwise fail loudly here
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

LLGO_BIN="${1:-${LLGO_BIN:-/tmp/llgo-idf}}"

# Everything below is expressed against the demo root, so the script can be
# invoked from anywhere (the historical contract was "run it from the demo
# root"; that is now implied rather than required).
cd "$DIR/../.."

# 0a. Sync the shared scaffold into this project. These files are identical
#     across every demo (the IDF project shell, the runtime symbol providers,
#     the linker fragment, the partition table), so _demo/common/ holds the
#     single copy and each project receives a generated one - the same model
#     as the glue below, and the reason a demo adds no build plumbing of its
#     own. Generated files are gitignored.
REPO_ROOT="${REPO_ROOT:-$(cd "$DIR/../../../.." && pwd)}"

# Copies are content-guarded: writing a file with identical bytes still bumps
# its mtime, and for the CMake configure inputs below (CMakeLists.txt,
# partitions.csv, the ld fragment) that would re-run CMake - and the whole
# dependency re-solve with it - on every build.
copy_if_changed() {
    if [ ! -f "$2" ] || ! cmp -s "$1" "$2"; then
        cp "$1" "$2"
    fi
}

for pair in "llgo_syms.S:host/llgo_syms.S" \
            "app_main.c:main/app_main.c" \
            "llgo_dram.ld:main/llgo_dram.ld" \
            "partitions.csv:partitions.csv" \
            "idfproj-CMakeLists.txt:CMakeLists.txt"; do
    src="${pair%%:*}"
    dst="${pair#*:}"
    copy_if_changed "$COMMON/$src" "$DIR/../$dst"
done

# 0b. Sync the package glues into the main component.
for g in ${GLUES:-}; do
    copy_if_changed "$REPO_ROOT/pkg/esp/$g/glue/${g}_glue.c" "$MAIN/${g}_glue.c"
done
for pair in ${GLUE_MAP:-}; do
    src="${pair%%:*}"
    dst="${pair#*:}"
    copy_if_changed "$REPO_ROOT/pkg/esp/$src" "$MAIN/$dst"
done
if [[ -f "$DIR/post-sync.sh" ]]; then
    # shellcheck source=/dev/null
    . "$DIR/post-sync.sh"
fi

# 1. Go -> relocatable object.
# shellcheck disable=SC2086
"$LLGO_BIN" build -target esp32s31-idf -o "$MAIN/llgo_tmp" ${LLGO_EXTRA_ARGS:-} ./goapp
mv "$MAIN/llgo_tmp.elf" "$MAIN/llgo_out.raw.o"

# 2. Assemble the runtime symbol providers and merge them in. The static GC
# window defaults to 64 KiB; the collector grows into PSRAM from there
# (tinygogc/gc_segment_idf.go), so the window only covers hot traffic on a
# PSRAM board. Raise GC_WIN_KB for a PSRAM-less board whose live set does not
# fit the default.
# LLGO_GC_WINDOW_KB still overrides from the environment, as it always did;
# otherwise the demo's artifact.conf value (or 64 KiB) applies.
GC_WIN_KB="${LLGO_GC_WINDOW_KB:-${GC_WIN_KB:-64}}"
# GO_TASK_STACK=1 (artifact.conf) assembles the dedicated Go-task stack
# (_llgo_go_task_stack, below the GC window) and app_main then spawns
# "llgo-go" on it - required when Go owns the FPU and higher-priority IDF
# tasks (WiFi) can preempt it (the coprocessor save-area check in
# freertos port.c).
GO_TASK_DEFINE=""
if [[ "${GO_TASK_STACK:-0}" == "1" ]]; then
    GO_TASK_DEFINE="-DLLGO_GO_TASK_STACK=1"
fi
"$CLANG" --target=riscv32-esp-elf -march=rv32imc_zicsr \
    -DLLGO_GC_WINDOW_BYTES=$((GC_WIN_KB * 1024)) $GO_TASK_DEFINE \
    -c "$COMMON/llgo_syms.S" -o "$DIR/llgo_syms.o"
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

# 4. Present the object as single-float ABI (see the header). Safe for this
# boundary: every Go<->C signature crossing the link passes integers only,
# Go's own soft-float helpers were renamed in step 3, and IDF's float paths
# stay on its own ilp32f libgcc.
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
for prefix in ${DIAG_SYMBOLS:-}; do
    echo "$prefix prefix (must appear above; NONE means the Go program does"
    echo "not import the package that owns it, or a synced glue is missing):"
    "$NM" -u "$OUT" | grep "$prefix" || echo "NONE"
done
