package gotest

import (
	"testing"
	"unsafe"
)

type nestedABIInner struct {
	A byte
	X int32
}

type nestedABIOuter struct {
	Prefix byte
	nestedABIInner
}

type nestedABIGeneric[T any] struct {
	Prefix T
	nestedABIInner
}

//go:noinline
func nestedABILayout(v nestedABIOuter) [3]uintptr {
	return [3]uintptr{unsafe.Sizeof(v), unsafe.Alignof(v), unsafe.Offsetof(v.X)}
}

//go:noinline
func nestedABIGenericLayout[T any](v nestedABIGeneric[T]) [3]uintptr {
	return [3]uintptr{unsafe.Sizeof(v), unsafe.Alignof(v), unsafe.Offsetof(v.X)}
}

//go:noinline
func nestedABIRoundTrip(v nestedABIOuter) nestedABIOuter {
	v.Prefix++
	v.A += 2
	v.X += 3
	return v
}

var nestedABIIndirect = nestedABIRoundTrip

type nestedABIFloats struct {
	Prefix byte
	Inner  struct{ X, Y float32 }
}

//go:noinline
func nestedABIFloatRoundTrip(v nestedABIFloats) nestedABIFloats {
	v.Prefix++
	v.Inner.X += 2
	v.Inner.Y += 3
	return v
}

func TestNestedAggregateABI(t *testing.T) {
	wantLayout := [3]uintptr{12, 4, 8}
	if got := nestedABILayout(nestedABIOuter{}); got != wantLayout {
		t.Fatalf("embedded layout = %v, want %v", got, wantLayout)
	}
	if got := nestedABIGenericLayout(nestedABIGeneric[byte]{}); got != wantLayout {
		t.Fatalf("generic embedded layout = %v, want %v", got, wantLayout)
	}
	input := nestedABIOuter{5, nestedABIInner{17, 100003}}
	want := nestedABIOuter{6, nestedABIInner{19, 100006}}
	if got := nestedABIRoundTrip(input); got != want {
		t.Fatalf("direct round trip = %v, want %v", got, want)
	}
	if got := nestedABIIndirect(input); got != want {
		t.Fatalf("indirect round trip = %v, want %v", got, want)
	}
	floats := nestedABIFloats{7, struct{ X, Y float32 }{1.5, -12.5}}
	wantFloats := nestedABIFloats{8, struct{ X, Y float32 }{3.5, -9.5}}
	if got := nestedABIFloatRoundTrip(floats); got != wantFloats {
		t.Fatalf("mixed integer/float round trip = %v, want %v", got, wantFloats)
	}
}
