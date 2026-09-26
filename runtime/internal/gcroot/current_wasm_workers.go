//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package gcroot

// The compiler addresses these fully qualified symbols directly, so their
// declarations and compiler-generated references must both remain native TLS.
// The address-bearing slots deliberately use uintptr: they describe the root
// machinery itself rather than Go heap references. Pointer-typing them would
// give the boundary helpers compiler root frames and splice those helpers into
// the chain they are switching.
//
//llgointernal:tls
var (
	currentRootChain uintptr
	sjljReplaying    bool
	activeContext    uintptr
	rebuilding       bool
)
