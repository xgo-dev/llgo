//go:build !llgo || !js || !wasm || !llgo.wasm.gc.linear || !llgo.wasm.workers

package gcroot

func lockRegistry()   {}
func unlockRegistry() {}
