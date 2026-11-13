package gotest

import (
	"reflect"
	"structs"
	"testing"
)

type onePtr struct{ P *int }

func wrapOnePtr(p *int) onePtr { return onePtr{P: p} }

func TestReflectCallOnePointerStruct(t *testing.T) {
	x := 7
	p := &x
	out := reflect.ValueOf(wrapOnePtr).Call([]reflect.Value{reflect.ValueOf(p)})
	got := out[0].Interface().(onePtr)
	if got.P != p {
		t.Fatalf("P = %p, want %p", got.P, p)
	}
	if *got.P != 7 {
		t.Fatalf("*P = %d, want 7", *got.P)
	}
}

func TestReflectCallOnePointerStructMultiReturn(t *testing.T) {
	x := 7
	p := &x
	type nested struct{ Inner onePtr }

	checkCall := func(name string, fn any, wantByte byte) {
		t.Helper()
		out := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(p)})
		if got := out[0].Uint(); got != uint64(wantByte) {
			t.Fatalf("%s first = %d, want %d", name, got, wantByte)
		}
		switch got := out[1].Interface().(type) {
		case onePtr:
			if got.P != p {
				t.Fatalf("%s P = %p, want %p", name, got.P, p)
			}
		case nested:
			if got.Inner.P != p {
				t.Fatalf("%s Inner.P = %p, want %p", name, got.Inner.P, p)
			}
		default:
			t.Fatalf("%s unexpected result %T", name, got)
		}
	}

	checkCall("(byte, One)", func(p *int) (byte, onePtr) { return 23, onePtr{P: p} }, 23)
	checkCall("(uint32, One)", func(p *int) (uint32, onePtr) { return 23, onePtr{P: p} }, 23)
	checkCall("(byte, nested)", func(p *int) (byte, nested) { return 23, nested{Inner: onePtr{P: p}} }, 23)

	typ := reflect.TypeOf((func(*int) (byte, onePtr))(nil))
	fn := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{
			reflect.ValueOf(byte(23)),
			reflect.ValueOf(onePtr{P: in[0].Interface().(*int)}),
		}
	})
	out := fn.Call([]reflect.Value{reflect.ValueOf(p)})
	if out[0].Interface().(byte) != 23 || out[1].Interface().(onePtr).P != p {
		t.Fatalf("MakeFunc Call = %v", out)
	}
	b, one := fn.Interface().(func(*int) (byte, onePtr))(p)
	if b != 23 || one.P != p {
		t.Fatalf("MakeFunc typed = (%d, %p), want (23, %p)", b, one.P, p)
	}
}

type hostOnePtr struct {
	_ structs.HostLayout
	P *int
}

type hostOnePtrAliasMarker = structs.HostLayout
type hostOnePtrDefinedMarker structs.HostLayout

type hostOnePtrAlias struct {
	_ hostOnePtrAliasMarker
	P *int
}

type hostOnePtrDefined struct {
	_ hostOnePtrDefinedMarker
	P *int
}

func wrapHostOnePtr(p *int) hostOnePtr           { return hostOnePtr{P: p} }
func takeHostOnePtr(h hostOnePtr) *int           { return h.P }
func wrapHostOnePtrAlias(p *int) hostOnePtrAlias { return hostOnePtrAlias{P: p} }
func wrapHostOnePtrDefined(p *int) hostOnePtrDefined {
	return hostOnePtrDefined{P: p}
}

//llgo:cdecl
type cdeclHostOnePtr struct {
	_ structs.HostLayout
	P *int
}

//llgo:cdecl
type cdeclZeroOnePtr struct {
	_ struct{}
	P *int
}

type cdeclHostOuter struct{ Inner cdeclHostOnePtr }
type cdeclHostArrayOuter struct{ Inner [1]cdeclHostOnePtr }

func wrapCdeclHostOnePtr(p *int) cdeclHostOnePtr { return cdeclHostOnePtr{P: p} }
func takeCdeclHostOnePtr(h cdeclHostOnePtr) *int { return h.P }
func wrapCdeclZeroOnePtr(p *int) cdeclZeroOnePtr { return cdeclZeroOnePtr{P: p} }
func wrapCdeclHostOuter(p *int) cdeclHostOuter {
	return cdeclHostOuter{Inner: cdeclHostOnePtr{P: p}}
}
func wrapCdeclHostArrayOuter(p *int) cdeclHostArrayOuter {
	return cdeclHostArrayOuter{Inner: [1]cdeclHostOnePtr{{P: p}}}
}

func TestReflectCallHostLayoutOnePointerStruct(t *testing.T) {
	x := 7
	p := &x

	out := reflect.ValueOf(wrapHostOnePtr).Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(hostOnePtr); got.P != p {
		t.Fatalf("return P = %p, want %p", got.P, p)
	}
	out = reflect.ValueOf(takeHostOnePtr).Call([]reflect.Value{reflect.ValueOf(hostOnePtr{P: p})})
	if got := out[0].Interface().(*int); got != p {
		t.Fatalf("arg P = %p, want %p", got, p)
	}

	for _, fn := range []any{wrapHostOnePtrAlias, wrapHostOnePtrDefined} {
		out = reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(p)})
		switch got := out[0].Interface().(type) {
		case hostOnePtrAlias:
			if got.P != p {
				t.Fatalf("alias P = %p, want %p", got.P, p)
			}
		case hostOnePtrDefined:
			if got.P != p {
				t.Fatalf("defined P = %p, want %p", got.P, p)
			}
		default:
			t.Fatalf("unexpected HostLayout result %T", got)
		}
	}

	typ := reflect.TypeOf((func(*int) hostOnePtr)(nil))
	fn := reflect.MakeFunc(typ, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(hostOnePtr{P: in[0].Interface().(*int)})}
	})
	out = fn.Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(hostOnePtr); got.P != p {
		t.Fatalf("MakeFunc Call P = %p, want %p", got.P, p)
	}
	if got := fn.Interface().(func(*int) hostOnePtr)(p); got.P != p {
		t.Fatalf("MakeFunc typed P = %p, want %p", got.P, p)
	}

	argTyp := reflect.TypeOf((func(hostOnePtr) *int)(nil))
	argFn := reflect.MakeFunc(argTyp, func(in []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(in[0].Interface().(hostOnePtr).P)}
	})
	out = argFn.Call([]reflect.Value{reflect.ValueOf(hostOnePtr{P: p})})
	if got := out[0].Interface().(*int); got != p {
		t.Fatalf("MakeFunc arg Call P = %p, want %p", got, p)
	}
	if got := argFn.Interface().(func(hostOnePtr) *int)(hostOnePtr{P: p}); got != p {
		t.Fatalf("MakeFunc arg typed P = %p, want %p", got, p)
	}

	type outer struct {
		_   structs.HostLayout
		Tag byte
		V   onePtr
	}
	out = reflect.ValueOf(func(p *int) (byte, outer) {
		return 23, outer{Tag: 9, V: onePtr{P: p}}
	}).Call([]reflect.Value{reflect.ValueOf(p)})
	if out[0].Interface().(byte) != 23 {
		t.Fatalf("outer first = %v, want 23", out[0].Interface())
	}
	got := out[1].Interface().(outer)
	if got.Tag != 9 || got.V.P != p {
		t.Fatalf("outer = %+v, want Tag=9 P=%p", got, p)
	}

	hType := reflect.StructOf([]reflect.StructField{
		{Name: "M", Type: reflect.TypeOf(structs.HostLayout{})},
		{Name: "P", Type: reflect.TypeOf((*int)(nil))},
	})
	fnType := reflect.FuncOf([]reflect.Type{reflect.TypeOf((*int)(nil))}, []reflect.Type{hType}, false)
	dyn := reflect.MakeFunc(fnType, func(in []reflect.Value) []reflect.Value {
		ret := reflect.New(hType).Elem()
		ret.Field(1).Set(in[0])
		return []reflect.Value{ret}
	})
	out = dyn.Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Field(1).Interface().(*int); got != p {
		t.Fatalf("StructOf MakeFunc P = %p, want %p", got, p)
	}
}

func TestReflectCallCdeclOnePointerStruct(t *testing.T) {
	x := 7
	p := &x

	out := reflect.ValueOf(wrapCdeclHostOnePtr).Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(cdeclHostOnePtr); got.P != p {
		t.Fatalf("cdecl HostLayout return P = %p, want %p", got.P, p)
	}
	out = reflect.ValueOf(takeCdeclHostOnePtr).Call([]reflect.Value{reflect.ValueOf(cdeclHostOnePtr{P: p})})
	if got := out[0].Interface().(*int); got != p {
		t.Fatalf("cdecl HostLayout arg P = %p, want %p", got, p)
	}
	out = reflect.ValueOf(wrapCdeclZeroOnePtr).Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(cdeclZeroOnePtr); got.P != p {
		t.Fatalf("cdecl zero-field return P = %p, want %p", got.P, p)
	}
	out = reflect.ValueOf(wrapCdeclHostOuter).Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(cdeclHostOuter); got.Inner.P != p {
		t.Fatalf("cdecl nested return P = %p, want %p", got.Inner.P, p)
	}
	out = reflect.ValueOf(wrapCdeclHostArrayOuter).Call([]reflect.Value{reflect.ValueOf(p)})
	if got := out[0].Interface().(cdeclHostArrayOuter); got.Inner[0].P != p {
		t.Fatalf("cdecl nested array return P = %p, want %p", got.Inner[0].P, p)
	}
}
