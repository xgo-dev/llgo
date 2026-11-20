//go:build llgo && wasm && wasip1

package runtime

import _ "unsafe"

const decimalPrintLLGoFiles = "; _wrap/print_decimal_wasi.c"

//go:linkname PrintInt C.llgo_print_int
func PrintInt(value int64)

//go:linkname PrintUint C.llgo_print_uint
func PrintUint(value uint64)
