//go:build !llgo || !wasm || !llgo.wasm.gc.linear

package gcroot

var (
	currentRootChain uintptr
	sjljReplaying    bool
	activeContext    uintptr
	rebuilding       bool
)
