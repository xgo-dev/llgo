//go:build llgo

package reflect_test

import (
	"reflect"
	"structs"
	"testing"
	"unsafe"
)

//llgo:cdecl
type hostLayoutCallback func(int32) int32

type hostLayoutCallbacks struct {
	_     structs.HostLayout
	Fn    hostLayoutCallback
	Items [2]hostLayoutCallback
	Tail  uint32
}

//go:noinline
func clearHostLayoutCallback(fn *hostLayoutCallback) { *fn = nil }

func TestHostLayoutCallbackStorage(t *testing.T) {
	// HostLayout must preserve the existing addressable Go callback storage.
	// On profiled wasm32 this is wider than a native function pointer slot.
	type ordinary struct {
		Fn    hostLayoutCallback
		Items [2]hostLayoutCallback
		Tail  uint32
	}
	var plain ordinary
	var values [2]hostLayoutCallbacks
	h := &values[0]
	if unsafe.Sizeof(*h) != unsafe.Sizeof(plain) ||
		unsafe.Offsetof(h.Items) != unsafe.Offsetof(plain.Items) ||
		unsafe.Offsetof(h.Tail) != unsafe.Offsetof(plain.Tail) ||
		uintptr(unsafe.Pointer(&values[1]))-uintptr(unsafe.Pointer(h)) != unsafe.Sizeof(plain) {
		t.Fatal("HostLayout changed callback storage or array stride")
	}
	typ := reflect.TypeOf(*h)
	if typ.Size() != unsafe.Sizeof(*h) || typ.Field(2).Offset != unsafe.Offsetof(h.Items) || typ.Field(3).Offset != unsafe.Offsetof(h.Tail) {
		t.Fatal("HostLayout metadata disagrees with callback storage")
	}
	add := func(x int32) int32 { return x + 1 }
	*h = hostLayoutCallbacks{Fn: add, Items: [2]hostLayoutCallback{add, add}, Tail: 0xdeadbeef}
	clearHostLayoutCallback(&h.Fn)
	clearHostLayoutCallback(&h.Items[0])
	if h.Fn != nil || h.Items[0] != nil || h.Items[1](41) != 42 || h.Tail != 0xdeadbeef {
		t.Fatal("callback address escape overwrote adjacent storage")
	}
	h.Fn = add
	reflect.ValueOf(h).Elem().FieldByName("Fn").SetZero()
	if h.Fn != nil || h.Items[0] != nil || h.Items[1](41) != 42 || h.Tail != 0xdeadbeef {
		t.Fatal("callback reflection clear overwrote adjacent storage")
	}
}
