//go:build llgo && js && wasm && llgo.wasm.emscripten && !llgo.wasm.emscripten.memory64

package emscripten

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

// wasiCIovec is shared directly with the host's fd_write import.
type wasiCIovec struct {
	_    structs.HostLayout
	data uint32
	size uint32
}

//go:wasmimport wasi_snapshot_preview1 fd_write
func wasiFDWrite(fd, iov, count, written uint32) uint32

func writeStderr(data c.Pointer, size uintptr) {
	if uintptr(uint32(size)) != size {
		panic("emscripten: console write exceeds wasm32 size_t")
	}

	iov := wasiCIovec{data: uint32(uintptr(data)), size: uint32(size)}
	var written uint32
	wasiFDWrite(
		2,
		uint32(uintptr(unsafe.Pointer(&iov))),
		1,
		uint32(uintptr(unsafe.Pointer(&written))),
	)
}
