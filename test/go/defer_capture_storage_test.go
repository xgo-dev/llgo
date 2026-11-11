package gotest

import (
	"runtime"
	"testing"
)

//go:noinline
func deferredCaptureResult(value int, callback func()) (result int) {
	returned, shared := false, value
	defer func() {
		if recover() != nil {
			if returned {
				panic("callback returned before panic")
			}
			result = shared
		} else {
			if !returned {
				panic("callback did not return")
			}
			result += shared
		}
	}()
	defer func() { shared += 7 }()
	callback()
	returned = true
	shared += 11
	return shared
}

func TestDeferCaptureStorageReturnAndPanic(t *testing.T) {
	if got := deferredCaptureResult(5, func() {}); got != 39 {
		t.Fatalf("normal return = %d, want 39", got)
	}
	if got := deferredCaptureResult(5, func() { panic("callback") }); got != 12 {
		t.Fatalf("recovered return = %d, want 12", got)
	}
}

//go:noinline
func deferredCapturePanicMutation() (result int) {
	value := 1
	defer func() { recover(); result = value }()
	defer func() { value = 9; panic("deferred mutation") }()
	return 3
}

//go:noinline
func deferredCaptureRecursion(n int) (result int) {
	value := n
	defer func() { result += value }()
	if n == 0 {
		return 1
	}
	result = deferredCaptureRecursion(n - 1)
	value += 10
	return
}

//go:noinline
func escapedDeferredCapture() func() int {
	value := 1
	defer func() { value = 9 }()
	return func() int { value++; return value }
}

func TestDeferCaptureStorageIdentity(t *testing.T) {
	if got := deferredCapturePanicMutation(); got != 9 {
		t.Fatalf("deferred panic mutation = %d, want 9", got)
	}
	if got := deferredCaptureRecursion(3); got != 37 {
		t.Fatalf("recursive result = %d, want 37", got)
	}
	f := escapedDeferredCapture()
	if first, second := f(), f(); first != 10 || second != 11 {
		t.Fatalf("escaped capture = %d, %d, want 10, 11", first, second)
	}
}

type deferCapturePayload struct {
	value int
	pad   [256]byte
}

//go:noinline
func newDeferCapturePayload(value int) *deferCapturePayload {
	return &deferCapturePayload{value: value}
}

//go:noinline
func deferredCapturePointer(value int, callback func()) (result int) {
	p := newDeferCapturePayload(value)
	defer func() { result = p.value }()
	callback()
	p = newDeferCapturePayload(value + 1)
	runtime.GC()
	return
}

func TestDeferCaptureStorageGC(t *testing.T) {
	for i := 0; i < 8; i++ {
		if got := deferredCapturePointer(i, runtime.GC); got != i+1 {
			t.Fatalf("pointer capture = %d, want %d", got, i+1)
		}
	}
}

func TestDeferCaptureStorageGoexit(t *testing.T) {
	done := make(chan int, 1)
	go func() {
		value := 1
		defer func() { done <- value }()
		defer func() { value = 9 }()
		runtime.Goexit()
	}()
	if got := <-done; got != 9 {
		t.Fatalf("Goexit capture = %d, want 9", got)
	}
}
