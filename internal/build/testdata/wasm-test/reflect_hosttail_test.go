package wasmtest

import (
	"reflect"
	"structs"
	"testing"
)

// Keep in sync with test/go/reflect_call_hosttail_test.go.

type hostTail struct {
	_ structs.HostLayout
	P *int
	Z struct{}
}

type hostGoTail struct {
	Value hostTail
	Tail  struct{}
}

func hostTailPair(p *int) (hostTail, byte) { return hostTail{P: p}, 23 }

func hostGoTailPair(p *int) (hostGoTail, byte) {
	return hostGoTail{Value: hostTail{P: p}}, 23
}

func TestReflectCallHostLayoutTail(t *testing.T) {
	checkHostLayoutTailCall(t, hostTailPair, func(v hostTail) *int { return v.P })
}

func TestReflectCallHostLayoutNestedTail(t *testing.T) {
	checkHostLayoutTailCall(t, hostGoTailPair, func(v hostGoTail) *int { return v.Value.P })
}

func checkHostLayoutTailCall[T any](t *testing.T, pair func(*int) (T, byte), ptr func(T) *int) {
	t.Helper()
	x := 7
	p := &x
	out := reflect.ValueOf(pair).Call([]reflect.Value{reflect.ValueOf(p)})
	got := out[0].Interface().(T)
	if ptr(got) != p || out[1].Uint() != 23 {
		t.Fatalf("Call: P=%p want=%p byte=%d", ptr(got), p, out[1].Uint())
	}

	fn := reflect.MakeFunc(reflect.TypeOf(pair), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(got), reflect.ValueOf(byte(23))}
	}).Interface().(func(*int) (T, byte))
	got, tag := fn(p)
	if ptr(got) != p || tag != 23 {
		t.Fatalf("MakeFunc: P=%p want=%p byte=%d", ptr(got), p, tag)
	}
}
