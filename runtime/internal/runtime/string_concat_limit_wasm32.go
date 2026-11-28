//go:build wasm && !llgo.wasm.emscripten.memory64

package runtime

// Wasm32 keeps 64-bit Go lengths and uintptr values over 32-bit addresses.
// Reject sizes that would narrow at the allocator or memcpy boundary before
// allocating or copying. maxAlloc alone follows the wider Go word size.
const stringConcatMaxAlloc = 1<<32 - 1
