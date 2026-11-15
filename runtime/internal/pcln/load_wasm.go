//go:build llgo && llgo_pclntab_external && wasm

package pcln

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"unsafe"
)

const externalPCLNGOARCH = 3

//go:linkname nativePointerSize C.llgo_pcln_pointer_size
func nativePointerSize() uint32

func externalBuffer(ptr unsafe.Pointer, size uint32) []byte {
	if ptr == nil {
		return nil
	}
	if size < uint32(externalPCLNHeaderSize) || size > uint32(externalPCLNMaxSize) {
		c.Free(ptr)
		return nil
	}
	return unsafe.Slice((*byte)(ptr), int(size))
}
