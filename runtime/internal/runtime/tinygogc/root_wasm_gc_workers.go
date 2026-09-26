//go:build llgo && wasm && llgo.wasm.gc.linear && !nogc && !baremetal && ((js && llgo.wasm.workers) || (wasip1 && llgo.wasi_threads))

package tinygogc

import "unsafe"

// Other workers can still have conservative references to a retired root.
// Unlink it now and let the next synchronized collection reclaim its storage.
func releaseRootStorage(unsafe.Pointer) {}
