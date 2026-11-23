//go:build baremetal && !nogc && llgo.baremetal.stdio

package runtime

import "unsafe"

// SetFinalizer has no dispatch on bare metal. Targets that patch the
// standard library (llgo.baremetal.stdio) fail loudly instead of silently
// dropping a registered finalizer.
func SetFinalizer(_ any, finalizer any) {
	if finalizer != nil {
		panic("runtime.SetFinalizer: no finalizer dispatch on bare metal")
	}
}

// llgoSetFinalizerPointer is the compiler-emitted fast path of SetFinalizer
// (cl only emits the call on wasm). The panic keeps a future gate relaxation
// from turning this combination into an undefined symbol at link time.
func llgoSetFinalizerPointer(_, _ unsafe.Pointer, _ uintptr) {
	panic("runtime.llgoSetFinalizerPointer: no finalizer dispatch on bare metal")
}
