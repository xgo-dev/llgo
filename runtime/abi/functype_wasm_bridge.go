//go:build llgo && wasm && wasip1

package abi

import "unsafe"

// FuncType carries compiler-generated entries because the WASI host cannot
// supply libffi closures or dynamically typed indirect calls. JavaScript and
// Emscripten profiles retain the smaller libffi descriptor layout.
type FuncType struct {
	Type
	In    []*Type
	Out   []*Type
	Call_ unsafe.Pointer
	Make_ unsafe.Pointer
}
