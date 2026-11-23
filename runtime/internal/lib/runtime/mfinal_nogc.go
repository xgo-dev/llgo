//go:build nogc && (!wasm || !llgo.wasm.gc.linear)

package runtime

import "unsafe"

// SetFinalizer is a no-op when garbage collection is disabled because objects
// are never discovered as unreachable.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}

func llgoSetFinalizerPointer(_, _ unsafe.Pointer, _ uintptr) {}
