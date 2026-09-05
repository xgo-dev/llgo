//go:build wasm && llgo.wasm.gc.linear

package runtime

// SetFinalizer is not implemented by the initial WebAssembly collector.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}
