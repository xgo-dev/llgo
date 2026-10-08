//go:build llgo && js && wasm && llgo.wasm.workers

// Package wasmworkers provides an explicit way to start independent work in
// LLGo's bounded browser worker pool.
package wasmworkers

import llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"

// GoIndependent starts fn on a scheduler worker chosen from the configured
// pool. syscall/js values may be shared: their operations run in the shared
// Go main worker's JavaScript realm. The closure must not carry raw Emscripten
// handles or thread-local C state from its caller.
func GoIndependent(fn func()) {
	llruntime.SpawnIndependentWasmG(fn)
}
