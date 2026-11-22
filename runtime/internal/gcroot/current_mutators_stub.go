//go:build !llgo || !wasip1 || !wasm || !llgo.wasm.gc.linear || !llgo.wasi_threads

package gcroot

func concurrentWASIMutators() bool { return true }
