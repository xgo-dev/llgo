package gotest

import (
	"reflect"
	"structs"
	"testing"
)

type hostTail struct {
	_ structs.HostLayout
	P *int
	Z struct{}
}

func hostTailPair(p *int) (hostTail, byte) { return hostTail{P: p}, 23 }

func TestReflectCallHostLayoutTail(t *testing.T) {
	x := 7
	p := &x
	out := reflect.ValueOf(hostTailPair).Call([]reflect.Value{reflect.ValueOf(p)})
	got := out[0].Interface().(hostTail)
	if got.P != p || out[1].Uint() != 23 {
		t.Fatalf("Call: P=%p want=%p byte=%d", got.P, p, out[1].Uint())
	}

	fn := reflect.MakeFunc(reflect.TypeOf(hostTailPair), func([]reflect.Value) []reflect.Value {
		return []reflect.Value{
			reflect.ValueOf(hostTail{P: p}),
			reflect.ValueOf(byte(23)),
		}
	}).Interface().(func(*int) (hostTail, byte))
	got, tag := fn(p)
	if got.P != p || tag != 23 {
		t.Fatalf("MakeFunc: P=%p want=%p byte=%d", got.P, p, tag)
	}
}
