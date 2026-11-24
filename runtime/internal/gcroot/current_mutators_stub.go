//go:build !llgo || !wasip1 || !wasm || !llgo.wasm.gc.linear || !llgo.wasi_threads || !llgo.wasi.mutatorselect

package gcroot

const WASISelectableMutators = false

func WASIConcurrentMutators() bool { return true }
