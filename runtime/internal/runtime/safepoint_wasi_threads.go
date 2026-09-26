//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package runtime

// CooperativeSafepoint is inserted at Go function entries and loop backs.
// The pthread owning this G parks only when a collector requests a stop.
func CooperativeSafepoint() {
	wasiGCSafepoint()
}
