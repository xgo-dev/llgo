//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

import _ "unsafe"

func gcStopWorld() {
	wasmStopWorld()
}

func gcResumeWorld() {
	wasmResumeWorld()
}

//go:linkname wasmStopWorld github.com/xgo-dev/llgo/runtime/internal/runtime.wasmGCStopTheWorld
func wasmStopWorld()

//go:linkname wasmResumeWorld github.com/xgo-dev/llgo/runtime/internal/runtime.wasmGCResumeWorld
func wasmResumeWorld()
