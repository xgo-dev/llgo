//go:build llgo && js && wasm && llgo.wasm.emscripten && llgo.wasm.emscripten.memory64

package emscripten

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/structs"
)

// wasiCIovec is shared directly with the host's fd_write import.
type wasiCIovec struct {
	_    structs.HostLayout
	data uint64
	size uint64
}

//go:wasmimport wasi_snapshot_preview1 fd_write
func wasiFDWrite(fd uint32, iov, count, written uint64) uint32

func writeStderr(data c.Pointer, size uintptr) {
	iov := wasiCIovec{data: uint64(uintptr(data)), size: uint64(size)}
	var written uint64
	wasiFDWrite(
		2,
		uint64(uintptr(unsafe.Pointer(&iov))),
		1,
		uint64(uintptr(unsafe.Pointer(&written))),
	)
}
