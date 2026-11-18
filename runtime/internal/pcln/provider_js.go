//go:build llgo && llgo_pclntab_external && wasm && js

package pcln

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
	"unsafe"
)

const externalPCLNGOOS = 4
const LLGoFiles = "_wrap/load_js.c"

//go:linkname startLoad C.llgo_pcln_load_start
func startLoad(request unsafe.Pointer)

func readExternalPCLN(poll func()) []byte {
	if poll == nil {
		return nil
	}
	request := c.Calloc(1, 16)
	if request == nil {
		return nil
	}
	defer c.Free(request)
	startLoad(request)
	for atomic.Load((*uint32)(request)) == 0 {
		poll()
	}
	size := *(*uint32)(unsafe.Add(request, 4))
	ptr := *(*unsafe.Pointer)(unsafe.Add(request, 8))
	return externalBuffer(ptr, size)
}
