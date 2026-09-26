package cgo_test

import (
	"runtime/cgo"
	"testing"
)

func TestHandleLifecycle(t *testing.T) {
	type payload struct{ name string }
	want := &payload{name: "llgo"}
	handle := cgo.NewHandle(want)
	if got, ok := handle.Value().(*payload); !ok || got != want {
		t.Fatalf("Value = %#v, want the original payload", got)
	}
	handle.Delete()
	if panicValue := panicFrom(func() { handle.Value() }); panicValue == nil {
		t.Fatal("Value on a deleted handle did not panic")
	}
}

func panicFrom(f func()) (value any) {
	defer func() {
		value = recover()
	}()
	f()
	return nil
}
