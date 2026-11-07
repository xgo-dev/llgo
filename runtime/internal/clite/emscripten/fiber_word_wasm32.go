//go:build llgo && wasm && !llgo.wasm.emscripten.memory64

package emscripten

// emscripten_fiber_t has eight physical pointer-sized slots. Use an explicit
// storage width: HostLayout must not narrow ordinary addressable Go uintptrs.
type fiberWord = uint32
