package gotest

import (
	"runtime"
	"testing"
	"unsafe"
)

type uintptrEscapesObject struct {
	value int
	// Avoid depending on the collector's tiny-object/finalizer packing.
	padding [32]byte
}

//go:noinline
func newUintptrEscapesObject(value int) unsafe.Pointer {
	obj := &uintptrEscapesObject{value: value}
	runtime.SetFinalizer(obj, func(obj *uintptrEscapesObject) { obj.value = -1 })
	return unsafe.Pointer(obj)
}

//go:noinline
//go:uintptrescapes
func checkUintptrEscapes(want int, first, second uintptr, rest ...uintptr) {
	runtime.GC()
	runtime.GC()
	if (*uintptrEscapesObject)(unsafe.Pointer(first)).value != want {
		panic("uintptrescapes: first object was finalized")
	}
	if (*uintptrEscapesObject)(unsafe.Pointer(second)).value != want {
		panic("uintptrescapes: second object was finalized")
	}
	for _, ptr := range rest {
		if (*uintptrEscapesObject)(unsafe.Pointer(ptr)).value != want {
			panic("uintptrescapes: variadic object was finalized")
		}
	}
}

//go:noinline
//go:uintptrescapes
func forwardUintptrEscapes(want int, first, second uintptr, rest ...uintptr) {
	runtime.GC()
	checkUintptrEscapes(want, first, second, rest...)
}

//go:noinline
func laterUintptrEscapesArgument(value int) unsafe.Pointer {
	// The preceding argument's pointer-to-uintptr conversion has already
	// happened, but the marked callee has not been entered yet.
	runtime.GC()
	runtime.GC()
	return newUintptrEscapesObject(value)
}

func TestUintptrEscapesCalls(t *testing.T) {
	checkUintptrEscapes(11,
		uintptr(newUintptrEscapesObject(11)), uintptr(laterUintptrEscapesArgument(11)),
		uintptr(newUintptrEscapesObject(11)), uintptr(laterUintptrEscapesArgument(11)))
	forwardUintptrEscapes(23,
		uintptr(newUintptrEscapesObject(23)), uintptr(newUintptrEscapesObject(23)),
		uintptr(newUintptrEscapesObject(23)), uintptr(newUintptrEscapesObject(23)))
}

func TestUintptrEscapesDeferredCalls(t *testing.T) {
	func() {
		defer checkUintptrEscapes(31,
			uintptr(newUintptrEscapesObject(31)), uintptr(newUintptrEscapesObject(31)),
			uintptr(newUintptrEscapesObject(31)))
		runtime.GC()
	}()
	func() {
		// Each iteration overwrites the caller's temporary root slots. Older
		// arguments must remain reachable through the deferred-call records.
		for i := 0; i < 8; i++ {
			defer checkUintptrEscapes(40+i,
				uintptr(newUintptrEscapesObject(40+i)), uintptr(newUintptrEscapesObject(40+i)),
				uintptr(newUintptrEscapesObject(40+i)))
		}
		runtime.GC()
	}()
}

//go:noinline
//go:uintptrescapes
func consumeUintptrEscapes(gate <-chan struct{}, done chan<- int, ptr uintptr) {
	<-gate
	runtime.GC()
	runtime.GC()
	done <- (*uintptrEscapesObject)(unsafe.Pointer(ptr)).value
}

//go:noinline
func startUintptrEscapes(gate <-chan struct{}, done chan<- int) {
	var local uintptrEscapesObject
	local.value = 71
	runtime.SetFinalizer(&local, func(obj *uintptrEscapesObject) { obj.value = -1 })
	go consumeUintptrEscapes(gate, done, uintptr(unsafe.Pointer(&local)))
}

func TestUintptrEscapesGoroutineOutlivesCaller(t *testing.T) {
	gate := make(chan struct{})
	done := make(chan int)
	startUintptrEscapes(gate, done)
	runtime.GC()
	runtime.GC()
	close(gate)
	if got := <-done; got != 71 {
		t.Fatalf("goroutine observed %d, want 71", got)
	}
}

type uintptrEscapesLeaf struct{}

//go:noinline
//go:uintptrescapes
func (uintptrEscapesLeaf) Check(first, second uintptr) {
	checkUintptrEscapes(83, first, second)
}

type uintptrEscapesOuter struct{ uintptrEscapesLeaf }

func TestUintptrEscapesMethodWrappers(t *testing.T) {
	uintptrEscapesOuter{}.Check(uintptr(newUintptrEscapesObject(83)), uintptr(laterUintptrEscapesArgument(83)))
	uintptrEscapesOuter.Check(uintptrEscapesOuter{}, uintptr(newUintptrEscapesObject(83)), uintptr(laterUintptrEscapesArgument(83)))
	// Go does not propagate the directive through a saved function value.
	// Preserve the ordinary unsafe conversion contract for this wrapper call.
	bound := uintptrEscapesOuter{}.Check
	first, second := newUintptrEscapesObject(83), newUintptrEscapesObject(83)
	bound(uintptr(first), uintptr(second))
	runtime.KeepAlive(first)
	runtime.KeepAlive(second)
	defer uintptrEscapesOuter{}.Check(uintptr(newUintptrEscapesObject(83)), uintptr(newUintptrEscapesObject(83)))
}
