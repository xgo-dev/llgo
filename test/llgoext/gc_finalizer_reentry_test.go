//go:build llgo && !wasm && !baremetal && !nogc

package llgoext

import (
	"runtime"
	"testing"
	"unsafe"
)

//llgo:cdecl
type gcNativeFinalizer func(unsafe.Pointer, unsafe.Pointer)

//go:linkname gcReentryAlloc C.GC_malloc
func gcReentryAlloc(size uintptr) unsafe.Pointer

//go:linkname gcReentryRegister C.GC_register_finalizer_no_order
func gcReentryRegister(object unsafe.Pointer, fn gcNativeFinalizer, data unsafe.Pointer, oldFn, oldData *unsafe.Pointer)

//go:linkname gcReentryGetDemand C.GC_get_finalize_on_demand
func gcReentryGetDemand() int32

//go:linkname gcReentrySetDemand C.GC_set_finalize_on_demand
func gcReentrySetDemand(int32)

var gcNativeReentryDone chan struct{}

func gcNativeReenter(_, _ unsafe.Pointer) {
	// BDWGC invokes this C finalizer synchronously during the collection
	// batch. Both physical-thread and cross-goroutine reentry must progress.
	runtime.GC()
	done := make(chan struct{})
	go func() {
		runtime.GC()
		close(done)
	}()
	<-done
	close(gcNativeReentryDone)
}

func TestRuntimeGCNativeFinalizerReentry(t *testing.T) {
	previous := gcReentryGetDemand()
	gcReentrySetDemand(1)
	t.Cleanup(func() { gcReentrySetDemand(previous) })
	gcNativeReentryDone = make(chan struct{})
	created := make(chan struct{})
	go func() {
		object := gcReentryAlloc(4096)
		gcReentryRegister(object, gcNativeReenter, nil, nil, nil)
		close(created)
	}()
	<-created
	for range 64 {
		runtime.Gosched()
		runtime.GC()
		select {
		case <-gcNativeReentryDone:
			return
		default:
		}
	}
	t.Fatal("native C finalizer did not complete nested GC requests")
}
