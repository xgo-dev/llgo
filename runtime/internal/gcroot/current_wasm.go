//go:build llgo && wasm && llgo.wasm.gc.linear && !llgo.wasm.workers

package gcroot

import _ "unsafe"

// Keep the chain head pointer-free so its boundary helpers do not acquire a
// compiler root frame while they are changing the chain itself.
//
//go:linkname currentRootChain llvm_gc_root_chain
var currentRootChain uintptr

//go:linkname sjljReplaying llvm_gc_root_sjlj_replaying
var sjljReplaying bool

var activeContext uintptr
var rebuilding bool
