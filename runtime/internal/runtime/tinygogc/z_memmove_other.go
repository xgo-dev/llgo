//go:build !tinygo.riscv32

package tinygogc

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Portable fallbacks so gc_tinygo.go compiles unchanged on non-RV32 targets;
// every tinygo.riscv32 build (currently all baremetal) reroutes to the
// word-block copies.
func llgoMemmove(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer {
	c.Memmove(dst, src, n)
	return dst
}

func llgoMemcpy(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer {
	c.Memcpy(dst, src, n)
	return dst
}
