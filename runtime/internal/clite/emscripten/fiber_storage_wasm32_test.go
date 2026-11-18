//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package emscripten

// Host storage has four-byte pointer slots even with eight-byte Go words.
const expectedFiberSize = uintptr(8 * 4)
