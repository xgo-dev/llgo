package maps

import (
	"internal/abi"
	"unsafe"
)

//go:linkname fatal github.com/goplus/llgo/runtime/internal/runtime.fatal
func fatal(s string)

//go:linkname rand github.com/goplus/llgo/runtime/internal/runtime.fastrand64
func rand() uint64

//go:linkname typedmemmove github.com/goplus/llgo/runtime/internal/runtime.Typedmemmove
func typedmemmove(typ *abi.Type, dst, src unsafe.Pointer)

//go:linkname typedmemclr github.com/goplus/llgo/runtime/internal/runtime.Typedmemclr
func typedmemclr(typ *abi.Type, ptr unsafe.Pointer)

//go:linkname newobject github.com/goplus/llgo/runtime/internal/runtime.newobject
func newobject(typ *abi.Type) unsafe.Pointer

//go:linkname newarray github.com/goplus/llgo/runtime/internal/runtime.newarray
func newarray(typ *abi.Type, n int) unsafe.Pointer
