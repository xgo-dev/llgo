//go:build !llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear

package runtime

func registerWasiGCThread()   {}
func unregisterWasiGCThread() {}

const wasiSelectMainContext = false

func wasiMainG() *g { return nil }
