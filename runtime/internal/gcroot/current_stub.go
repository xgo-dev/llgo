//go:build !llgo || !wasm || !llgo_wasm_gc

package gcroot

import "unsafe"

var currentRootChain unsafe.Pointer
