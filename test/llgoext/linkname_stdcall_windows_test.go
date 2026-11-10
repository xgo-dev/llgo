//go:build llgo && windows

package llgoext

import "testing"

func TestLinknameStdcall(t *testing.T) {
	if &stdcallCounter != &nativeCounter {
		t.Fatal("stdcall scalar must reference ordinary native storage")
	}
	if got := invokeStdcallEntry(35); got != 42 {
		t.Fatalf("export = %d", got)
	}
	if got := stdcallPlaceholder(21); got != 42 {
		t.Fatalf("placeholder = %d", got)
	}
	if got := stdcallCallback(21); got != 42 {
		t.Fatalf("loaded pointer = %d", got)
	}
	var asGo func(int32) int32 = stdcallCallback
	if got := asGo(21); got != 42 {
		t.Fatalf("Go funcval conversion = %d", got)
	}
	// A Go closure cannot be stored back into a native function-pointer slot.
	t.Cleanup(func() { stdcallCallback = stdcallTwice })
	stdcallCallback = nil
	if stdcallCallback != nil {
		t.Fatal("nil function pointer not preserved")
	}
}
