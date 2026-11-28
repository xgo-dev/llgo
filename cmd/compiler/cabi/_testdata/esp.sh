#!/bin/sh
mkdir -p arch/esp32
mkdir -p arch/esp32c3
(cd arch/esp32 && clang -target xtensa-esp-unknown-elf -S -emit-llvm ../../wrap/*.c)
(cd arch/esp32c3 && clang -target riscv32-esp-unknown-elf -S -emit-llvm ../../wrap/*.c)
