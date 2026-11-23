//go:build !tinygo.riscv32

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Portable fallbacks for the memory copies z_slice.go uses. Every
// tinygo.riscv32 build (currently all baremetal) reroutes to its C word-block
// implementations in _wrap/memmove_riscv32.c (z_memmove_riscv32.go); every
// other target keeps the libc functions unchanged.
func llgoMemmove(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer {
	c.Memmove(dst, src, n)
	return dst
}

func llgoMemcpy(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer {
	c.Memcpy(dst, src, n)
	return dst
}
