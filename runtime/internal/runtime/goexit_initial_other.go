//go:build !llgo || !wasm || !wasip1 || !llgo.wasi_threads

package runtime

func parkInitialWasiThread(*g) {}
