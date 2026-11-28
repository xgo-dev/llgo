#!/bin/sh
mkdir -p arch/amd64
mkdir -p arch/arm64
mkdir -p arch/i386
mkdir -p arch/armv6
mkdir -p arch/riscv64
mkdir -p arch/riscv64_lp64d
mkdir -p arch/riscv64_lp64f
mkdir -p arch/wasm32
mkdir -p arch/riscv32_ilp32
mkdir -p arch/riscv32_ilp32f
mkdir -p arch/riscv32_ilp32d
(cd arch/amd64 && clang -target amd64-unknown-linux-gnu -S -emit-llvm ../../wrap/*.c)
(cd arch/arm64 && clang -target aarch64-linux-gnu  -S -emit-llvm ../../wrap/*.c)
(cd arch/i386 && clang -target i386-unknown-linux-gnu  -S -emit-llvm ../../wrap/*.c)
(cd arch/armv6 && clang -target armv6l-unknown-linux-gnueabihf  -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv64 && clang -target riscv64-unknown-elf  -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv64_lp64f && clang -target riscv64-unknown-elf -mabi=lp64f -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv64_lp64d && clang -target riscv64-unknown-elf -mabi=lp64d -S -emit-llvm ../../wrap/*.c)
(cd arch/wasm32 && clang -target wasm32-unknown-emscripten  -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv32_ilp32 && clang -target riscv32-unknown-elf -mabi=ilp32  -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv32_ilp32f && clang -target riscv32-unknown-elf -mabi=ilp32f  -S -emit-llvm ../../wrap/*.c)
(cd arch/riscv32_ilp32d && clang -target riscv32-unknown-elf -mabi=ilp32d  -S -emit-llvm ../../wrap/*.c)
