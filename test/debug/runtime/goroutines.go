package main

import (
	"runtime"
	"time"
	"unsafe"
)

const goroutineWaiting = 4 // _Gwaiting in runtime/internal/runtime/runtime2.go.

// Use the native snapshot API rather than walking the live debugger registry.
// runtime.Stack infers channel wait reasons from frames, even before the
// worker publishes the waiting state that the debugger actually reads.
//
//go:linkname captureThreads C.llgo_traceback_capture
func captureThreads(except uint64) unsafe.Pointer

//go:linkname nextThread C.llgo_traceback_next
func nextThread(snapshot unsafe.Pointer) unsafe.Pointer

//go:linkname threadInfo C.llgo_traceback_info
func threadInfo(snapshot unsafe.Pointer, id, parent *uint64, count *uintptr, state *uint32) *uintptr

//go:linkname freeThreads C.llgo_traceback_free
func freeThreads(snapshots unsafe.Pointer)

func parkedWorkers() int {
	snapshots := captureThreads(1)
	defer freeThreads(snapshots)
	waiting := 0
	for s := snapshots; s != nil; s = nextThread(s) {
		var id, parent uint64
		var count uintptr
		var state uint32
		threadInfo(s, &id, &parent, &count, &state)
		if parent == 1 && state == goroutineWaiting {
			waiting++
		}
	}
	return waiting
}

func waitForParkedWorkers() {
	deadline := time.Now().Add(5 * time.Second)
	for parkedWorkers() != 2 {
		if time.Now().After(deadline) {
			panic("workers did not park before the debugger breakpoint")
		}
		runtime.Gosched()
	}
}
