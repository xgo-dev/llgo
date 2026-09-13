//go:build !windows && !js

package ffi

import "github.com/xgo-dev/llgo/runtime/internal/clite/ffi"

func newComplexType(elem *Type, size uintptr, align uint16) *Type {
	return newAggregateType(size, align, ffi.Complex, []*Type{elem, nil})
}
