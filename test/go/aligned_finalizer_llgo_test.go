//go:build llgo && (amd64 || arm64) && !nogc

package gotest

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
)

//go:linkname nativeGCBase C.GC_base
func nativeGCBase(p unsafe.Pointer) unsafe.Pointer

//go:linkname nativeGCFinalizer C.GC_register_finalizer
func nativeGCFinalizer(p, fn, data unsafe.Pointer, oldFn, oldData *unsafe.Pointer)

//llgo:cdecl
type nativeFinalizerCallback func(unsafe.Pointer, unsafe.Pointer)

func nativeInteriorAllocation(t *testing.T) (p, base unsafe.Pointer) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		p = alignedAllocation(37, 64)
		base = nativeGCBase(p)
		if base != nil && base != p {
			return
		}
	}
	t.Fatal("no interior aligned allocation")
	return
}

func TestRuntimeAlignedFinalizer(t *testing.T) {
	for _, asInterface := range []bool{false, true} {
		// Choose a logical object whose aligned address is inside its GC block.
		p, base := nativeInteriorAllocation(t)
		obj := (*[37]byte)(p)
		obj[0] = 93
		done := make(chan uintptr, 1)
		if asInterface {
			runtime.SetFinalizer(obj, func(v any) {
				q := v.(*[37]byte)
				if q[0] != 93 {
					done <- 0
					return
				}
				done <- uintptr(unsafe.Pointer(q))
			})
		} else {
			runtime.SetFinalizer(obj, func(q *[37]byte) {
				if q[0] != 93 {
					done <- 0
					return
				}
				done <- uintptr(unsafe.Pointer(q))
			})
		}
		// Read back and remove the actual collector registration, then invoke its
		// callback deterministically. This checks the pointer delivered to Go
		// without depending on conservative-stack retention or GC timing.
		var fn, data unsafe.Pointer
		nativeGCFinalizer(base, nil, nil, &fn, &data)
		if fn == nil {
			runtime.SetFinalizer(obj, nil)
			t.Fatal("aligned object finalizer not registered on its GC allocation")
		}
		callback := *(*nativeFinalizerCallback)(unsafe.Pointer(&fn))
		callback(base, data)
		select {
		case got := <-done:
			if got != uintptr(p) {
				t.Fatalf("finalizer pointer = %#x, want %#x", got, uintptr(p))
			}
		case <-time.After(5 * time.Second):
			t.Fatal("finalizer callback did not run")
		}
		runtime.KeepAlive(obj)
	}
}

func TestRuntimeAlignedCleanup(t *testing.T) {
	p, base := nativeInteriorAllocation(t)
	obj := (*[37]byte)(p)
	done := make(chan int, 1)
	cleanup := runtime.AddCleanup(obj, func(value int) { done <- value }, 93)
	var fn, data unsafe.Pointer
	nativeGCFinalizer(base, nil, nil, &fn, &data)
	if fn == nil {
		cleanup.Stop()
		t.Fatal("aligned object cleanup not registered on its GC allocation")
	}
	callback := *(*nativeFinalizerCallback)(unsafe.Pointer(&fn))
	callback(base, data)
	select {
	case got := <-done:
		if got != 93 {
			t.Fatalf("cleanup argument = %d, want 93", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup callback did not run")
	}
	runtime.KeepAlive(obj)
}
