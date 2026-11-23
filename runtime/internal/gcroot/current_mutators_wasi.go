//go:build llgo && wasip1 && wasm && llgo.wasm.gc.linear && llgo.wasi_threads && llgo.wasi.mutatorselect

package gcroot

import _ "unsafe"

const WASISelectableMutators = true

// The wasi-libc ABI remains threaded. The final link can select just one
// Go mutator when the executable has no supported way to start another.
//
//go:linkname wasiGCConcurrent C.__llgo_wasi_gc_concurrent
var wasiGCConcurrent uint8

func WASIConcurrentMutators() bool { return wasiGCConcurrent != 0 }
