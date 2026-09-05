//go:build (nogc && (!wasm || !llgo.wasm.gc.linear)) || (llgo_noffi && !nogc && !wasm)

package runtime

// SetFinalizer is a no-op when garbage collection is disabled or when the
// native runtime is built without libffi. Typed finalizers use SetFinalizerPtr
// in the latter case.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}

func runFinalizers() {}
