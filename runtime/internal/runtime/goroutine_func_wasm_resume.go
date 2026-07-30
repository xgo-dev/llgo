//go:build llgo && wasm && llgo.wasm_resume && (js || wasip1) && !(wasip1 && llgo.wasi_threads)

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/wasmresume"
)

// goroutineFunc is a compiler-generated start entry for one resumable G.
//
//llgo:type C
type goroutineFunc func(*wasmresume.Context, unsafe.Pointer) *wasmresume.Frame
