//go:build !llgo || !js || !wasm || !llgo.wasm.gc.linear || !llgo.wasm.workers

package tinygogc

func gcStopWorld()   {}
func gcResumeWorld() {}
