# Vendored riscv32 libffi

Prebuilt static archive for RV32 baremetal targets (rv32imc, ilp32,
soft-float), used by reflect's native call bridge (`Value.Call`, `Method`)
on the ESP32-C3/S31 series.

- Upstream: [libffi](https://sourceware.org/libffi/) 3.4.7
  (`include/ffi.h`, `src/riscv/ffitarget.h`)
- Target: `riscv32-esp-elf` (`-march=rv32imc -mabi=ilp32`, soft float)
- Objects: `src/prep_cif.c`, `src/types.c`, `src/riscv/ffi.c`,
  `src/riscv/sysv.S`
- S31 closures use the executable DIRAM-backed tinygogc allocation instead
  of `closures.c`'s OS-backed executable mappings. Reflection retains the
  closure allocation, CIF and callback state through its `keepAlive` roots.
- On S31, `--wrap=__clear_cache` directs libffi trampoline synchronization to
  a local `fence.i` implementation. This is a current-hart operation; it does
  not implement cross-hart instruction-cache synchronization. The wrapper is
  selected only by `tinygo.riscv32 && esp32s31`; other targets retain their
  existing cache-clear behavior.

Rebuild against the ESP Clang and newlib sysroot that llgo caches under
`~/Library/Caches/llgo/crosscompile`, then replace `libffi.a` and the
generated headers:

```sh
CLANG=$CACHE/esp-clang-*/bin/clang++
NEWLIB=$CACHE/newlib-esp32-*/newlib/libc/include
for f in src/prep_cif.c src/types.c src/riscv/ffi.c; do
  $CLANG --target=riscv32-esp-elf -march=rv32imc -mabi=ilp32 -mno-relax \
    -Oz -fno-builtin -ffreestanding -ffunction-sections -fdata-sections \
    -isystem $NEWLIB -I include -I src -I <fficonfig dir> -x c $f -c
done
$CLANG --target=riscv32-esp-elf -march=rv32imc -mabi=ilp32 -mno-relax \
  -I include -I <fficonfig asm dir> \
  -x assembler-with-cpp src/riscv/sysv.S -c
llvm-ar rcs libffi.a prep_cif.o types.o ffi.o sysv.o
```

`<fficonfig dir>/fficonfig.h` must define `FFI_HIDDEN` and
`HAVE_LONG_DOUBLE_VARIANT 0` (plus `#include <string.h>` for the C objects);
the assembly pass uses `.hidden` instead of the attribute and no libc
includes.
