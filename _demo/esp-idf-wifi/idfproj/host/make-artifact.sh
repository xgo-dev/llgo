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
#
# Usage: ./make-artifact.sh [path-to-llgo] (default /tmp/llgo-idf)
set -euo pipefail

LLGO_BIN="${1:-/tmp/llgo-idf}"
LLGO_ROOT="${LLGO_ROOT:-/Users/ma6174/gopath/src/github.com/goplus/llgo}"
DIR="$(cd "$(dirname "$0")" && pwd)"
MAIN="$DIR/../main"
OUT="$MAIN/llgo_out.o"

NM=nm
OBJCOPY=objcopy
CLANG=clang
LLD=ld.lld
if [[ -x "$HOME/Library/Caches/llgo/crosscompile/esp-clang-22.1.4_20261030-aarch64-apple-darwin/bin/clang" ]]; then
    ESPBIN="$HOME/Library/Caches/llgo/crosscompile/esp-clang-22.1.4_20261030-aarch64-apple-darwin/bin"
    CLANG="$ESPBIN/clang"
    LLD="$ESPBIN/ld.lld"
    NM="$ESPBIN/llvm-nm"
    OBJCOPY="$ESPBIN/llvm-objcopy"
fi

cd "$DIR/../.."

# 1. Go -> relocatable object.
"$LLGO_BIN" build -target esp32s31-idf -o "$MAIN/llgo_tmp" ./goapp
mv "$MAIN/llgo_tmp.elf" "$MAIN/llgo_out.raw.o"

# 2. Assemble the runtime symbol providers and merge them in.
"$CLANG" --target=riscv32-esp-elf -march=rv32imc_zicsr -c "$DIR/llgo_syms.S" -o "$DIR/llgo_syms.o"
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
# soft-float ilp32, while IDF compiles esp32s31 with the FPU (ilp32f) and
# GNU ld hard-fails the EF_RISCV_FLOAT_ABI mismatch at the final link.
# Marking the object as float-abi=single is safe for this boundary: every
# Go<->C signature crossing the link (main, vTaskDelay, xTaskGetTickCount)
# passes integers only, Go's own soft-float helpers were renamed in step 3,
# and IDF's float paths stay on its own ilp32f libgcc.
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
echo "artifact written: $OUT ($(stat -f %z "$OUT") bytes)"
echo "undefined symbols remaining (all must resolve at the IDF link):"
"$NM" -u "$OUT"
