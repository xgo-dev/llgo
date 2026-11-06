//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package abi

import "unsafe"

// FuncType includes optional compiler-generated call entries for layouts that
// libffi cannot represent, such as packed aggregate parameters and results.
type FuncType struct {
	Type
	In    []*Type
	Out   []*Type
	Call_ unsafe.Pointer
	Make_ unsafe.Pointer
}
