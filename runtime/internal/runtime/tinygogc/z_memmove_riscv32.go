//go:build tinygo.riscv32

package tinygogc

import "unsafe"

// Go wrappers for the C word-block memory copies that the runtime package
// compiles from _wrap/memmove_riscv32.c into its archive. gc_tinygo.go calls
// these instead of the libc c.Memmove/c.Memcpy, whose newlib-esp32 RV32
// builds are byte loops. The C functions allocate nothing, never yield, and
// stay inside a single hart, so they are safe in allocation and heap-growth
// paths. The unique llgo symbol names resolve against the runtime archive at
// final link, mirroring the z_memmove_riscv32.go wrappers in the runtime.
//
//go:linkname llgoMemmove C.llgoMemmove
func llgoMemmove(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer

//go:linkname llgoMemcpy C.llgoMemcpy
func llgoMemcpy(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer
