//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear && (!llgo || !js || !llgo.wasm.workers))

package tinygogc

import "unsafe"

// Single-owner collectors retain the historical behavior: the active stack
// or scheduler owns the returned allocation, so no separate root registry is
// needed. Release the allocation eagerly once that owner retires it.
func AllocRoot(size uintptr) unsafe.Pointer { return Alloc(size) }

func FreeRoot(ptr unsafe.Pointer) { Free(ptr) }
