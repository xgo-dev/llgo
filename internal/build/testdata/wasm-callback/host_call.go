//go:build js && wasm

package main

import "syscall/js"

// The host harness supplies a function that grows this module's memory or
// exits it. This supplements the ordinary Go-compatible syscall/js tests
// without exposing module internals in the production host runner.
func testHostCallBoundary(probe js.Value) {
	mode := js.Global().Get("llgoHostMode").String()
	expected := js.Global().Get("llgoHostExpected")
	defer func() {
		p := recover()
		if mode == "throw" {
			err, ok := p.(js.Error)
			if !ok || !err.Value.Equal(expected) {
				panic("memory growth lost the original JavaScript exception")
			}
			println("wasm host call boundary ok")
			return
		}
		if p != nil {
			panic("host control signal became a Go panic")
		}
	}()
	var result js.Value
	switch js.Global().Get("llgoHostOperation").String() {
	case "Call":
		object := js.Global().Get("Object").New()
		object.Set("probe", probe)
		result = object.Call("probe", expected)
	case "Invoke":
		result = probe.Invoke(expected)
	case "New":
		result = probe.New(expected)
	default:
		panic("unknown host call operation")
	}
	if mode != "return" {
		panic("host exception or exit unexpectedly returned")
	}
	if !result.Equal(expected) {
		panic("memory growth lost the JavaScript result")
	}
	println("wasm host call boundary ok")
}
