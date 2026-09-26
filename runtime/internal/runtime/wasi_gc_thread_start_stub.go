//go:build !windows && (!llgo || !wasm || (wasip1 && llgo.wasi_threads)) && (!llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear)

package runtime

import "unsafe"

func wasiGCThreadStart(arg unsafe.Pointer) unsafe.Pointer { return mstart(arg) }
