package runtime

import (
	"unsafe"
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	llrt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func addOff(ptr unsafe.Pointer, off int32) unsafe.Pointer {
	return unsafe.Pointer(uintptr(int64(uintptr(ptr)) + int64(off)))
}

//go:linkname reflect_resolveNameOff reflect.resolveNameOff
func reflect_resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer {
	return addOff(ptrInModule, off)
}

//go:linkname reflect_resolveTypeOff reflect.resolveTypeOff
func reflect_resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer {
	return addOff(rtype, off)
}

//go:linkname reflectlite_resolveNameOff internal/reflectlite.resolveNameOff
func reflectlite_resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer {
	return addOff(ptrInModule, off)
}

//go:linkname reflectlite_resolveTypeOff internal/reflectlite.resolveTypeOff
func reflectlite_resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer {
	return addOff(rtype, off)
}

//go:linkname llrtIfaceE2I github.com/xgo-dev/llgo/runtime/internal/runtime.IfaceE2I
func llrtIfaceE2I(t *abi.Type, src any, dst unsafe.Pointer)

//go:linkname reflectlite_unsafe_New internal/reflectlite.unsafe_New
func reflectlite_unsafe_New(t *abi.Type) unsafe.Pointer {
	return llrt.New(t)
}

//go:linkname reflectlite_ifaceE2I internal/reflectlite.ifaceE2I
func reflectlite_ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	llrtIfaceE2I(t, src, dst)
}

//go:linkname reflectlite_typedmemmove internal/reflectlite.typedmemmove
func reflectlite_typedmemmove(t *abi.Type, dst, src unsafe.Pointer) {
	llrt.Typedmemmove(t, dst, src)
}
