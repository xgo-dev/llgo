//go:build baremetal

package runtime

import "unsafe"

// The tinygogc finalizer bridge is only wired for WebAssembly. Baremetal
// builds still provide the entry points so finalizer calls type-check and
// link; registration is a no-op until that bridge is ported.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}

func llgoSetFinalizerPointer(objPtr, fn unsafe.Pointer, resultSize uintptr) {
	_, _, _ = objPtr, fn, resultSize
}
