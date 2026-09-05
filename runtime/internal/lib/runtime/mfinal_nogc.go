//go:build nogc && (!wasm || !llgo.wasm.gc.linear)

package runtime

// SetFinalizer is a no-op when garbage collection is disabled because objects
// are never discovered as unreachable.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}
