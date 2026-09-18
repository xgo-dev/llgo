package weak_test

import (
	"runtime"
	"testing"
	"time"
	"weak"
)

func TestMake(t *testing.T) {
	x := new(int)
	*x = 42

	wp := weak.Make(x)
	if wp == (weak.Pointer[int]{}) {
		t.Error("Make returned zero value")
	}

	val := wp.Value()
	if val == nil {
		t.Error("Value() returned nil for live object")
	}
	if *val != 42 {
		t.Errorf("*Value() = %d, want 42", *val)
	}
}

func TestPointerValue(t *testing.T) {
	x := new(string)
	*x = "hello"

	wp := weak.Make(x)
	val := wp.Value()
	if val == nil {
		t.Fatal("Value() returned nil")
	}
	if *val != "hello" {
		t.Errorf("*Value() = %q, want hello", *val)
	}
}

func TestPointerGC(t *testing.T) {
	wp := makeCollectableWeakPointer(123, nil)

	deadline := time.Now().Add(3 * time.Second)
	for !weakPointerCleared(wp) && time.Now().Before(deadline) {
		runtime.GC()
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	if val := wp.Value(); val != nil {
		t.Fatalf("weak pointer remained valid after its object became unreachable: %v", *val)
	}
}

func TestPointerGCWithCleanup(t *testing.T) {
	cleaned := make(chan struct{}, 1)
	wp := makeCollectableWeakPointer(456, func(x *int) {
		runtime.AddCleanup(x, func(struct{}) {
			cleaned <- struct{}{}
		}, struct{}{})
	})

	deadline := time.Now().Add(3 * time.Second)
	for !weakPointerCleared(wp) && time.Now().Before(deadline) {
		runtime.GC()
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	if val := wp.Value(); val != nil {
		t.Fatalf("weak pointer remained valid after its object became unreachable: %v", *val)
	}
	select {
	case <-cleaned:
	case <-time.After(3 * time.Second):
		t.Fatal("cleanup did not run after the weak pointer became nil")
	}
}

func makeCollectableWeakPointer(value int, register func(*int)) weak.Pointer[int] {
	result := make(chan weak.Pointer[int], 1)
	done := make(chan struct{})
	go func() {
		x := new(int)
		*x = value
		if register != nil {
			register(x)
		}
		result <- weak.Make(x)
		close(done)
	}()
	wp := <-result
	<-done
	return wp
}

//go:noinline
func weakPointerCleared(wp weak.Pointer[int]) bool {
	// Keep the temporary strong pointer out of the frame that starts the next
	// collection. Conservative collectors may otherwise retain its stale bits.
	return wp.Value() == nil
}

func TestPointerZeroValue(t *testing.T) {
	var wp weak.Pointer[int]

	val := wp.Value()
	if val != nil {
		t.Errorf("zero Pointer.Value() = %v, want nil", val)
	}
}

func TestPointerMultipleTypes(t *testing.T) {
	type MyStruct struct {
		Field int
	}

	s := &MyStruct{Field: 99}
	wp := weak.Make(s)

	val := wp.Value()
	if val == nil {
		t.Fatal("Value() returned nil")
	}
	if val.Field != 99 {
		t.Errorf("val.Field = %d, want 99", val.Field)
	}
}

func TestPointerNilInput(t *testing.T) {
	var nilPtr *int
	wp := weak.Make(nilPtr)

	val := wp.Value()
	if val != nil {
		t.Errorf("Make(nil).Value() = %v, want nil", val)
	}
}

func TestPointerIdentity(t *testing.T) {
	x, y := new([256]byte), new([256]byte)
	x[0], y[0] = 1, 2
	xw, yw := weak.Make(x), weak.Make(y)
	if xw != weak.Make(x) {
		t.Fatal("repeated Make returned a different handle for the same object")
	}
	if xw == yw {
		t.Fatal("Make returned the same handle for distinct objects")
	}
	runtime.GC()
	if xw != weak.Make(x) || yw != weak.Make(y) {
		t.Fatal("GC changed the identity of a live weak pointer")
	}
	if xw.Value() != x || yw.Value() != y {
		t.Fatal("GC invalidated a live weak pointer")
	}
	runtime.KeepAlive(x)
	runtime.KeepAlive(y)
}
