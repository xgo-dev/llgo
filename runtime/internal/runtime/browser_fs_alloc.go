//go:build llgo && js && wasm

package runtime

import "unsafe"

// Filesystem proxy buffers are held by JavaScript numeric addresses, which
// are invisible to the Go collector. Keep them rooted until the proxy returns.
// Doubles keep this JS-only boundary numeric in both memory widths.

//export llgo_browser_fs_malloc
func browserFSMalloc(size float64) float64 {
	return float64(uintptr(AllocRoot(uintptr(size))))
}

//export llgo_browser_fs_free
func browserFSFree(address float64) {
	FreeRoot(unsafe.Pointer(uintptr(address)))
}
