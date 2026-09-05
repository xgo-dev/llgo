//go:build !llgo || !wasm || !llgo.wasm.gc.linear

package gcroot

import "unsafe"

var (
	currentRootChain unsafe.Pointer
	sjljReplaying    bool
)
