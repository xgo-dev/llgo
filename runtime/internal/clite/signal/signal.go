//go:build !wasm

package signal

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

const (
	LLGoPackage = "link"
)

//llgo:cdecl
type SignalHandler func(c.Int)

type sigactiont struct {
	_       structs.HostLayout
	handler SignalHandler
	tramp   unsafe.Pointer
	mask    c.Int
	flags   c.Int
}

//go:linkname sigaction C.sigaction
func sigaction(sig c.Int, act, old *sigactiont) c.Int

func Signal(sig c.Int, hanlder SignalHandler) c.Int {
	var act sigactiont
	act.handler = hanlder
	return sigaction(sig, &act, nil)
}
