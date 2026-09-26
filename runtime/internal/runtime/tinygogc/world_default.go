//go:build !llgo || !wasm || !llgo.wasm.gc.linear || ((!js || !llgo.wasm.workers) && (!wasip1 || !llgo.wasi_threads))

package tinygogc

func gcStopWorld() bool { return true }
func gcResumeWorld()    {}
