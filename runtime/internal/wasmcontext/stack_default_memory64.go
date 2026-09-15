//go:build llgo && wasm && llgo.wasm.emscripten.memory64

package wasmcontext

// Memory64 doubles pointer slots and compiler GC roots. With only 128 KiB,
// standard-library ASN.1 and ML-DSA paths cross the fixed Fiber boundary.
const defaultStackSize = uintptr(256 << 10)
