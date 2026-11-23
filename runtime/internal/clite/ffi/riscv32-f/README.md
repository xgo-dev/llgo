# Vendored riscv32 libffi, ilp32f hard-float variant

Prebuilt static archive for the RV32 baremetal hard-float targets
(rv32imafc, ilp32f), used by reflect's native call bridge (`Value.Call`,
`Method`) on the ESP32-S31 -f target. The linker rejects mixing floating-point
ABIs, so this archive sits beside the ilp32 soft-float one in
`../riscv32` and is selected by the target's `ilp32f` ABI.

- Upstream: [libffi](https://sourceware.org/libffi/) 3.4.7
  (`include/ffi.h`, `src/riscv/ffitarget.h` — identical to the soft-float
  variant; the headers are ABI-independent)
- Target: `riscv32-esp-elf` (`-march=rv32imafc_zicsr_zifencei -mabi=ilp32f`)
- Objects: `src/prep_cif.c`, `src/types.c`, `src/riscv/ffi.c`,
  `src/riscv/sysv.S`
- Closure allocation and `--wrap=__clear_cache` behave exactly as described in
  `../riscv32/README.md`; only the ABI differs.

Rebuild against the ESP Clang and newlib sysroot that llgo caches under
`~/Library/Caches/llgo/crosscompile`, then replace `libffi.a`:

```sh
CLANG=$CACHE/esp-clang-*/bin/clang++
NEWLIB=$CACHE/newlib-esp32-*/newlib/libc/include
SOFT=../riscv32   # ABI-independent include headers + this recipe's baseline
SRC=<libffi-3.4.7 source>

CFLAGS="--target=riscv32-esp-elf -march=rv32imafc_zicsr_zifencei -mabi=ilp32f \
  -mno-relax -Oz -fno-builtin -ffreestanding -ffunction-sections \
  -fdata-sections -isystem $NEWLIB -I $SOFT/include -I $SRC/include \
  -I <fficonfig dir> -DSTDC_HEADERS=1"
for f in src/prep_cif.c src/types.c src/riscv/ffi.c; do
  $CLANG $CFLAGS -x c $f -c
done
$CLANG --target=riscv32-esp-elf -march=rv32imafc_zicsr_zifencei -mabi=ilp32f \
  -mno-relax -Oz -I <fficonfig-asm dir> \
  -x assembler-with-cpp src/riscv/sysv.S -c
llvm-ar rcs libffi.a prep_cif.o types.o ffi.o sysv.o
```

`<fficonfig dir>/fficonfig.h` must define `FFI_HIDDEN` and
`HAVE_LONG_DOUBLE_VARIANT 0` (plus `#include <string.h>` for the C objects).
Two details that differ from the soft-float build:

- `-DSTDC_HEADERS=1` is required: without it, libffi's own `ffi_common.h`
  falls back to `#define memcpy(d,s,n) bcopy((s),(d),(n))`, whose void return
  breaks `data = memcpy(...)` in `src/riscv/ffi.c`. (The soft-float recipe
  never hit this because its sysv.S assembly was the only share of the build
  that mattered for that historical archive.)
- The sysv.S assembler pass needs a stub header directory: `<fficonfig-asm
  dir>/ffi.h` defining only `FFI_TRAMPOLINE_SIZE 24` (no typedefs — the
  assembler consumes preprocessed tokens) and `<fficonfig-asm
  dir>/fficonfig.h` defining an empty `FFI_HIDDEN` (sysv.S uses `.hidden`
  directives itself).
