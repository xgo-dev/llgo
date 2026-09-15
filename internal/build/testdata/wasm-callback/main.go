//go:build js && wasm

package main

import "syscall/js"

func main() {
	if probe := js.Global().Get("llgoHostProbe"); !probe.IsUndefined() {
		testHostCallBoundary(probe)
		return
	}
	done := make(chan struct{})
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		close(done)
		return nil
	})
	defer callback.Release()

	// Do not install a Go timer: the scheduler must yield to the host solely
	// because a JavaScript callback source can make this channel runnable.
	js.Global().Call("setTimeout", callback, 0)
	<-done
	println("wasm callback-only wake ok")
}
