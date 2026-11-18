//go:build llgo

package gotest

import (
	"reflect"
	"structs"
	"testing"
	"unsafe"
)

type hostLayoutTail struct {
	_     structs.HostLayout
	Value uint64
	Tail  struct{}
}

func TestHostLayoutZeroTailAddress(t *testing.T) {
	var items [2]hostLayoutTail
	first := uintptr(unsafe.Pointer(&items[0]))
	next := uintptr(unsafe.Pointer(&items[1]))
	tail := uintptr(unsafe.Pointer(&items[0].Tail))
	if tail < first || tail >= next {
		t.Fatalf("tail address %x outside first element [%x, %x)", tail, first, next)
	}
	typ := reflect.TypeOf(items[0])
	if next-first != unsafe.Sizeof(items[0]) || typ.Size() != next-first || typ.Field(2).Offset != tail-first {
		t.Fatal("HostLayout source, storage, and reflection disagree")
	}
	// Copy and access fields after the padded element to exercise the LLVM
	// field-index mapping as well as array element stride.
	type outer struct {
		H     hostLayoutTail
		After uint64
	}
	a := outer{H: hostLayoutTail{Value: 7}, After: 9}
	b := a
	if b.H.Value != 7 || b.After != 9 {
		t.Fatal("padded HostLayout aggregate copy")
	}
}
