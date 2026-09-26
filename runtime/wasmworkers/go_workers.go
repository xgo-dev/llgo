//go:build llgo && js && wasm && llgo.wasm.workers

// Package wasmworkers provides an explicit way to start independent work in
// LLGo's bounded browser worker pool.
package wasmworkers

import llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"

// GoIndependent starts fn on a scheduler worker chosen from the configured
// pool. The closure must not carry syscall/js values, Emscripten handles, or
// thread-local C state from its caller. It may create its own values after it
// starts. An ordinary go statement inherits the caller's JavaScript realm
// after the caller has used syscall/js.
func GoIndependent(fn func()) {
	llruntime.SpawnIndependentWasmG(fn)
}
