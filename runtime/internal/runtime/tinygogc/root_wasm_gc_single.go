//go:build llgo && wasm && llgo.wasm.gc.linear && !nogc && !baremetal && !((js && llgo.wasm.workers) || (wasip1 && llgo.wasi_threads))

package tinygogc

import "unsafe"

// A single worker owns the retired Fiber or Asyncify stack exclusively. Free
// it before the scheduler resumes so stale stack words cannot become roots.
func releaseRootStorage(ptr unsafe.Pointer) { Free(ptr) }
