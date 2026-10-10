//go:build llgo

package llgoext

import _ "unsafe" // for go:linkname

// Use LLGoFiles instead of import "C" so the same C boundary runs on Wasm.
const LLGoFiles = "_wrap/variadic.c; _wrap/narrow.c"

//llgo:type C
type Variadic func(marker int32, __llgo_va_list ...any) int32

//go:linkname linkedVariadicFixed C.variadic_fixed
func linkedVariadicFixed(marker int32, value int32, number float64) int32

//go:linkname linkedVariadic C.variadic_indirect
func linkedVariadic(marker int32, __llgo_va_list ...any) int32

//go:linkname linkedVariadicAddress C.variadic_address
func linkedVariadicAddress() Variadic
