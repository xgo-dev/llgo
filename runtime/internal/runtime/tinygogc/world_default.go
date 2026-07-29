//go:build !llgo || !js || !wasm || !llgo_wasm_gc || !llgo.wasm_workers

package tinygogc

func gcStopWorld()   {}
func gcResumeWorld() {}
