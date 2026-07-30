//go:build llgo && wasm && llgo_wasm_gc

package gcroot

import "unsafe"

//go:linkname currentRootChain llvm_gc_root_chain
var currentRootChain unsafe.Pointer
