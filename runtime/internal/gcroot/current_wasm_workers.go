//go:build llgo && js && wasm && llgo_wasm_gc && llgo.wasm_workers

package gcroot

// The compiler uses the fully qualified currentRootChain symbol as its
// thread-local root-chain slot in multi-worker builds.
//
//llgo:tls
var (
	currentRootChain uintptr
	activeContext    uintptr
)
