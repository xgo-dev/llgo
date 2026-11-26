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
	if value < 0 {
		return -1
	}
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

//go:noinline
func deferredCaptureAggregate(callback func()) (result int) {
	var capture struct {
		payload *deferCapturePayload
		words   [13]uintptr
	}
	defer func() {
		recover()
		runtime.GC()
		result = capture.payload.value + int(capture.words[0]+capture.words[12])
	}()
	capture.payload = newDeferCapturePayload(7)
	capture.words[0] = 2
	callback()
	capture.words[12] = 3
	return
}

func TestDeferCaptureAggregateAcrossRecovery(t *testing.T) {
	if got := deferredCaptureAggregate(func() {}); got != 12 {
		t.Fatalf("normal aggregate capture = %d, want 12", got)
	}
	if got := deferredCaptureAggregate(func() { panic("callback") }); got != 9 {
		t.Fatalf("recovered aggregate capture = %d, want 9", got)
	}
}

//go:noinline
func deferredCaptureConditionalPointerAggregate(cond bool, callback func()) (result int) {
	if cond {
		var capture struct {
			payload *deferCapturePayload
			words   [6]uintptr
		}
		defer func() {
			recovered := recover() != nil
			runtime.GC()
			if capture.payload == nil {
				panic("conditional pointer aggregate root lost during GC")
			}
			result = capture.payload.value + int(capture.words[0]+capture.words[5])
			if recovered {
				result += 100
			}
		}()
		capture.payload = newDeferCapturePayload(13)
		capture.words[0] = 4
		callback()
		capture.words[5] = 5
		return
	}
	runtime.GC()
	return -7
}

func TestDeferCaptureConditionalBranchPointerAggregateGC(t *testing.T) {
	for i := 0; i < 4; i++ {
		if got := deferredCaptureConditionalPointerAggregate(false, func() {}); got != -7 {
			t.Fatalf("skipped branch = %d, want -7", got)
		}
		if got := deferredCaptureConditionalPointerAggregate(true, runtime.GC); got != 22 {
			t.Fatalf("conditional branch normal = %d, want 22", got)
		}
		if got := deferredCaptureConditionalPointerAggregate(true, func() {
			runtime.GC()
			panic("branch aggregate")
		}); got != 117 {
			t.Fatalf("conditional branch recovered = %d, want 117", got)
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
