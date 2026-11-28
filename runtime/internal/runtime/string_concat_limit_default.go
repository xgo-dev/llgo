//go:build !wasm || llgo.wasm.emscripten.memory64

package runtime

// Native Go words have the physical pointer width. Memory64 also uses a
// 64-bit address space, so the ordinary runtime allocation limit applies.
const stringConcatMaxAlloc = maxAlloc
