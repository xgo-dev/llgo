//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package abi

// FuncType represents a function type on targets that use libffi.
type FuncType struct {
	Type
	Types    **Type // input types followed by output types
	InCount  int
	OutCount int
}
