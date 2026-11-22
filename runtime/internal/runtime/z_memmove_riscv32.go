//go:build tinygo.riscv32

package runtime

import "unsafe"

// Go wrappers for the C word-block memory copies in
// _wrap/memmove_riscv32.c. z_slice.go calls these instead of the libc
// c.Memmove/c.Memcpy, whose newlib-esp32 RV32 builds are byte loops. The C
// functions use unique llgo names because the libc archives claim the
// memmove/memcpy symbol slots before package archives are scanned.
//
//go:linkname llgoMemmove C.llgoMemmove
func llgoMemmove(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer

//go:linkname llgoMemcpy C.llgoMemcpy
func llgoMemcpy(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer
