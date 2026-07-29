//go:build llgo && js && wasm && llgo_wasm_gc && llgo.wasm_workers

package tinygogc

import _ "unsafe"

func gcStopWorld() {
	wasmStopWorld()
}

func gcResumeWorld() {
	wasmResumeWorld()
}

//go:linkname wasmStopWorld github.com/goplus/llgo/runtime/internal/runtime.wasmGCStopTheWorld
func wasmStopWorld()

//go:linkname wasmResumeWorld github.com/goplus/llgo/runtime/internal/runtime.wasmGCResumeWorld
func wasmResumeWorld()
