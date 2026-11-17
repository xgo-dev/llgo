//go:build !wasm

package signal

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "link"
)

//llgo:cdecl
type SignalHandler func(c.Int)

// Signal installs a handler using the platform's sigaction layout. Hardware
// faults must remain deliverable after a panic longjmps out of the handler.
//
//go:linkname Signal C.llgo_core_signal
func Signal(sig c.Int, handler SignalHandler) c.Int
