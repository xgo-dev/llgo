//go:build llgo

package reflect_test

import (
	"reflect"
	"structs"
	"testing"
	"unsafe"
)

//llgo:cdecl
type hostCallback func(int32) int32

type hostCallbackRecord struct {
	_    structs.HostLayout
	Fn   hostCallback
	Tail uint32
}

type hostCallbackArray struct {
	_    structs.HostLayout
	Fn   [2]hostCallback
	Tail uint32
}

func hostCallbackAdd(x int32) int32 { return x + 1 }
func hostCallbackSub(x int32) int32 { return x - 1 }

//go:noinline
func setHostCallback(p *hostCallback, f hostCallback) { *p = f }

//go:noinline
func setHostCallbackElement(p *[2]hostCallback, f hostCallback) { p[1] = f }

func TestHostLayoutCallbackReflection(t *testing.T) {
	const sentinel = uint32(0xdeadbeef)
	t.Run("field", func(t *testing.T) {
		h := hostCallbackRecord{Fn: hostCallbackAdd, Tail: sentinel}
		setHostCallback(&h.Fn, hostCallbackSub)
		if h.Fn(42) != 41 || h.Tail != sentinel {
			t.Fatal("callback address store")
		}
		f := reflect.ValueOf(&h).Elem().FieldByName("Fn")
		f.SetZero()
		if h.Fn != nil || h.Tail != sentinel {
			t.Fatal("reflect.SetZero overwrote adjacent field")
		}
		f.Set(reflect.ValueOf(hostCallback(hostCallbackAdd)))
		if h.Fn(41) != 42 || h.Tail != sentinel {
			t.Fatal("reflected callback assignment")
		}
	})
	t.Run("array", func(t *testing.T) {
		h := hostCallbackArray{Fn: [2]hostCallback{hostCallbackAdd, hostCallbackAdd}, Tail: sentinel}
		setHostCallbackElement(&h.Fn, hostCallbackSub)
		if h.Fn[0](41) != 42 || h.Fn[1](42) != 41 || h.Tail != sentinel {
			t.Fatal("escaped callback array address")
		}
		f := reflect.ValueOf(&h).Elem().FieldByName("Fn")
		f.Index(0).SetZero()
		if h.Fn[0] != nil || h.Fn[1] == nil || h.Fn[1](42) != 41 || h.Tail != sentinel {
			t.Fatal("reflected array element stride")
		}
		f.Index(1).Set(reflect.ValueOf(hostCallback(hostCallbackAdd)))
		if h.Fn[1](41) != 42 || h.Tail != sentinel {
			t.Fatal("reflected array element assignment")
		}
		f.SetZero()
		if h.Fn[0] != nil || h.Fn[1] != nil || h.Tail != sentinel {
			t.Fatal("reflected callback array clear")
		}
	})
	t.Run("layout", func(t *testing.T) {
		var h hostCallbackRecord
		typ := reflect.TypeOf(h)
		callback := typ.Field(1).Type
		actual := uintptr(unsafe.Pointer(&h.Tail)) - uintptr(unsafe.Pointer(&h))
		if actual != unsafe.Sizeof(h.Fn) || callback.Size() != actual || callback.Align() != int(unsafe.Alignof(h.Fn)) || typ.Field(2).Offset != actual || typ.Size() != unsafe.Sizeof(h) {
			t.Fatal("callback source, storage and reflection disagree")
		}
		fields := []reflect.StructField{typ.Field(0), typ.Field(1), typ.Field(2)}
		dynamic := reflect.StructOf(fields)
		if dynamic.Size() != typ.Size() || dynamic.Align() != typ.Align() || dynamic.Field(2).Offset != typ.Field(2).Offset {
			t.Fatal("reflect.StructOf callback layout")
		}
	})
}

//llgo:cdecl
type legacyCallbackRecord struct {
	Fn   hostCallback
	Tail uint32
}

//llgo:cdecl
type legacyCallbackArray struct {
	Fn   [2]hostCallback
	Tail uint32
}

// No HostLayout marker is involved: preserve the original native ABI when a
// field address loses its containing-record context at a Go call boundary.
func TestLegacyCallbackAddressEscape(t *testing.T) {
	const sentinel = uint32(0xdeadbeef)
	h := legacyCallbackRecord{Fn: hostCallbackAdd, Tail: sentinel}
	setHostCallback(&h.Fn, nil)
	if h.Fn != nil || h.Tail != sentinel {
		t.Fatal("legacy callback pointer overwrote tail")
	}
	setHostCallback(&h.Fn, hostCallbackSub)
	if h.Fn(42) != 41 || h.Tail != sentinel {
		t.Fatal("legacy callback pointer assignment")
	}
	a := legacyCallbackArray{Fn: [2]hostCallback{hostCallbackAdd, hostCallbackAdd}, Tail: sentinel}
	setHostCallbackElement(&a.Fn, hostCallbackSub)
	if a.Fn[0](41) != 42 || a.Fn[1](42) != 41 || a.Tail != sentinel {
		t.Fatal("legacy callback array pointer stride")
	}
}

//llgo:cdecl
type nativeCallbackAlias = func(int32) int32

func TestNativeCallbackRepresentation(t *testing.T) {
	cb := hostCallback(hostCallbackAdd)
	typ := reflect.TypeOf(cb)
	if typ.Size() != unsafe.Sizeof(cb) || typ.Align() != int(unsafe.Alignof(cb)) {
		t.Fatal("native callback metadata")
	}
	in := []reflect.Value{reflect.ValueOf(int32(41))}
	h := hostCallbackRecord{Fn: cb, Tail: 0xdeadbeef}
	field := reflect.ValueOf(&h).Elem().FieldByName("Fn")
	if field.Call(in)[0].Int() != 42 || field.Interface().(hostCallback)(41) != 42 {
		t.Fatal("native callback treated as closure")
	}
	value := reflect.New(typ).Elem()
	value.Set(reflect.ValueOf(cb))
	if value.Call(in)[0].Int() != 42 {
		t.Fatal("allocated native callback")
	}
	value.SetZero()
	if !value.IsNil() {
		t.Fatal("allocated native callback clear")
	}
	array := reflect.ArrayOf(2, typ)
	if array.Size() != 2*unsafe.Sizeof(cb) {
		t.Fatal("dynamic native callback array size")
	}
	slice := reflect.MakeSlice(reflect.SliceOf(typ), 2, 2)
	slice.Index(0).Set(reflect.ValueOf(cb))
	slice.Index(1).Set(reflect.ValueOf(hostCallback(hostCallbackSub)))
	if slice.Slice(1, 2).Index(0).Call([]reflect.Value{reflect.ValueOf(int32(42))})[0].Int() != 41 {
		t.Fatal("dynamic callback slice stride")
	}
	// The same callback type also keeps this representation in ordinary Go
	// aggregates, including a direct-interface one-field wrapper.
	type plainRecord struct {
		Fn   [2]hostCallback
		Tail uint32
	}
	p := plainRecord{Fn: [2]hostCallback{cb, cb}, Tail: 0xdeadbeef}
	setHostCallbackElement(&p.Fn, hostCallbackSub)
	if p.Fn[0](41) != 42 || p.Fn[1](42) != 41 || p.Tail != 0xdeadbeef {
		t.Fatal("ordinary callback array address")
	}
	pt := reflect.TypeOf(p)
	if pt.Size() != unsafe.Sizeof(p) || pt.Field(1).Offset != unsafe.Offsetof(p.Tail) || unsafe.Offsetof(p.Tail) != 2*unsafe.Sizeof(cb) {
		t.Fatal("ordinary callback array layout")
	}
	type wrapper struct{ Fn hostCallback }
	w := wrapper{Fn: cb}
	wv := reflect.ValueOf(&w).Elem()
	wv.Set(reflect.ValueOf(wrapper{Fn: hostCallbackSub}))
	if wv.Interface().(wrapper).Fn(42) != 41 {
		t.Fatal("direct-interface callback wrapper")
	}
	dict := reflect.MakeMap(reflect.MapOf(reflect.TypeOf(0), typ))
	dict.SetMapIndex(reflect.ValueOf(1), reflect.ValueOf(cb))
	if dict.MapIndex(reflect.ValueOf(1)).Call(in)[0].Int() != 42 {
		t.Fatal("dynamic callback map")
	}
	type aliasRecord struct {
		_    structs.HostLayout
		Fn   nativeCallbackAlias
		Tail uint32
	}
	a := aliasRecord{Fn: hostCallbackAdd, Tail: 0xdeadbeef}
	av := reflect.ValueOf(&a).Elem().FieldByName("Fn")
	if av.Call(in)[0].Int() != 42 {
		t.Fatal("native alias call")
	}
	av.SetZero()
	if a.Fn != nil || a.Tail != 0xdeadbeef {
		t.Fatal("native alias clear")
	}
	captured := int32(40)
	goFunc := func(x int32) int32 { return x + captured }
	if reflect.ValueOf(goFunc).Call([]reflect.Value{reflect.ValueOf(int32(2))})[0].Int() != 42 {
		t.Fatal("Go closure environment")
	}
	if typ.ConvertibleTo(reflect.TypeOf(goFunc)) || reflect.TypeOf(goFunc).ConvertibleTo(typ) {
		t.Fatal("native callback/Go closure representation conflated")
	}
	t.Run("MakeFunc", func(t *testing.T) {
		defer func() {
			if got := recover(); got != "reflect.MakeFunc: native callback types are not supported" {
				t.Fatalf("MakeFunc native callback: %v", got)
			}
		}()
		reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value { return in })
	})
}
