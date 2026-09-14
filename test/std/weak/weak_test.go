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
	var wp weak.Pointer[int]

	func() {
		x := new(int)
		*x = 123
		wp = weak.Make(x)

		val := wp.Value()
		if val == nil || *val != 123 {
			t.Fatal("weak pointer should be valid before GC")
		}
	}()

	deadline := time.Now().Add(3 * time.Second)
	for wp.Value() != nil && time.Now().Before(deadline) {
		runtime.Gosched()
		runtime.GC()
		time.Sleep(time.Millisecond)
	}
	if val := wp.Value(); val != nil {
		t.Fatalf("weak pointer remained valid after its object became unreachable: %v", *val)
	}
}

func TestPointerGCWithCleanup(t *testing.T) {
	var wp weak.Pointer[int]
	cleaned := make(chan struct{}, 1)

	func() {
		x := new(int)
		*x = 456
		runtime.AddCleanup(x, func(struct{}) {
			cleaned <- struct{}{}
		}, struct{}{})
		wp = weak.Make(x)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for wp.Value() != nil && time.Now().Before(deadline) {
		runtime.Gosched()
		runtime.GC()
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
