//go:build !llgo || !wasm || !llgo.wasm.emscripten.memory64

package wasmcontext

// LLGo uses fixed-size Fiber stacks. 128 KiB is the smallest default that
// accommodates the standard-library depths exercised by the wasm acceptance
// suite on Memory32.
const defaultStackSize = uintptr(128 << 10)
