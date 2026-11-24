//go:build llgo && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package abi

import "unsafe"

// FuncType carries compiler-generated entries shared by all Wasm instances.
// WASI has no libffi closures, and Emscripten's dynamic function-table entries
// are local to the worker that installed them.
type FuncType struct {
	Type
	Types    **Type // input types followed by output types
	InCount  int
	OutCount int
	Call_    unsafe.Pointer
	Make_    unsafe.Pointer
}
