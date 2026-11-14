//go:build llgo && js && wasm && !llgo.wasm.debugger

package runtime

func captureWasmSuspension(*g) {}
