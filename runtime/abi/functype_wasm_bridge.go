//go:build llgo && wasm && (wasip1 || js)

package abi

import "unsafe"

// FuncType carries compiler-generated entries shared by all Wasm instances.
// WASI has no libffi closures, and Emscripten's dynamic function-table entries
// are local to the worker that installed them.
type FuncType struct {
	Type
	In    []*Type
	Out   []*Type
	Call_ unsafe.Pointer
	Make_ unsafe.Pointer
}
