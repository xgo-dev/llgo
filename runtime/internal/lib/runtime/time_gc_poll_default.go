//go:build !llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear

package runtime

const timerGCWaitQuantum = int64(0)

func timerGCSafepoint() {}
