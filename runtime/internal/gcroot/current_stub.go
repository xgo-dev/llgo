//go:build !llgo || !wasm || !llgo_wasm_gc

package gcroot

var (
	currentRootChain uintptr
	activeContext    uintptr
)
