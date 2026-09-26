//go:build !llgo || !wasip1 || !wasm || !llgo.wasi_threads

package runtime

func markTimerSystemG() {}
