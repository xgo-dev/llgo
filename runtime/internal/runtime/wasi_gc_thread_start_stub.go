//go:build !windows && (!llgo || !wasm || (wasip1 && llgo.wasi_threads)) && (!llgo || !wasip1 || !wasm || !llgo.wasi_threads || !llgo.wasm.gc.linear) && !(llgo && baremetal && tinygo.riscv32 && !nogc)

package runtime

import "unsafe"

func wasiGCThreadStart(arg unsafe.Pointer) unsafe.Pointer { return mstart(arg) }
