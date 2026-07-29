//go:build wasm && llgo_wasm_gc

package runtime

// SetFinalizer is not implemented by the initial WebAssembly collector.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}
