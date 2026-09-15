//go:build !llgo || !wasm || !wasip1

package abi

// FuncType represents a function type on targets that use libffi.
type FuncType struct {
	Type
	In  []*Type
	Out []*Type
}
