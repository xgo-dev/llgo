//go:build go1.23

package nativerepr

import (
	"structs"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/derived"
	"github.com/xgo-dev/llgo/test/llgoext/nativerepr/native"
)

func TestHostLayoutInheritedCallback(t *testing.T) {
	type Original struct {
		_        structs.HostLayout
		Callback native.Callback
		Tail     uint64
	}
	type Inherited struct {
		_        structs.HostLayout
		Callback derived.CallbackCopy
		Tail     uint64
	}
	var original Original
	var inherited Inherited
	if unsafe.Sizeof(original) != unsafe.Sizeof(inherited) || unsafe.Offsetof(original.Tail) != unsafe.Offsetof(inherited.Tail) {
		t.Fatal("HostLayout changed the inherited callback layout")
	}
	inherited.Callback = increment
	inherited.Tail = 123456
	if inherited.Callback(41) != 42 || inherited.Tail != 123456 {
		t.Fatal("HostLayout inherited callback call changed adjacent storage")
	}
	// HostLayout itself must keep the environment of an ordinary Go closure.
	marker := int32(40)
	host := struct {
		_        structs.HostLayout
		Callback func(int32) int32
	}{Callback: func(x int32) int32 { return marker + x }}
	if host.Callback(2) != 42 {
		t.Fatal("HostLayout lost an ordinary Go closure environment")
	}
}
