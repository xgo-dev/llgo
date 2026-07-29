//go:build llgo && wasm && llgo_wasm_gc && !llgo.wasm_workers

package gcroot

import _ "unsafe"

//go:linkname currentRootChain llvm_gc_root_chain
var currentRootChain uintptr

var activeContext uintptr
