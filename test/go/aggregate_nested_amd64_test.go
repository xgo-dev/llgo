//go:build !windows

package gotest

import "testing"

type nestedABIAligned[T int32 | float32] struct {
	_     [0]uint64
	Value T
}

type nestedABISeparated[T, U interface{ int32 | float32 }] struct {
	Low  T
	High nestedABIAligned[U]
}

//go:noinline
func nestedABISeparatedRoundTrip[T, U interface{ int32 | float32 }](v nestedABISeparated[T, U]) nestedABISeparated[T, U] {
	v.Low += 11
	v.High.Value += 13
	return v
}

var nestedABISeparatedIndirect = nestedABISeparatedRoundTrip[float32, int32]

//go:noinline
func nestedABIArrayRoundTrip(v [2]nestedABIAligned[int32]) [2]nestedABIAligned[int32] {
	v[0].Value += 11
	v[1].Value += 13
	return v
}

func TestNestedAggregateSeparatedRegisters(t *testing.T) {
	ints := nestedABISeparated[int32, int32]{Low: 7, High: nestedABIAligned[int32]{Value: 100003}}
	if got := nestedABISeparatedRoundTrip(ints); got.Low != 18 || got.High.Value != 100016 {
		t.Fatalf("separated integer fields corrupted: %+v", got)
	}
	floats := nestedABISeparated[float32, float32]{Low: 0.25, High: nestedABIAligned[float32]{Value: 100.5}}
	if got := nestedABISeparatedRoundTrip(floats); got.Low != 11.25 || got.High.Value != 113.5 {
		t.Fatalf("separated float fields corrupted: %+v", got)
	}
	mixed := nestedABISeparated[float32, int32]{Low: 0.25, High: nestedABIAligned[int32]{Value: 100003}}
	if got := nestedABISeparatedIndirect(mixed); got.Low != 11.25 || got.High.Value != 100016 {
		t.Fatalf("separated mixed fields corrupted through function value: %+v", got)
	}
	array := [2]nestedABIAligned[int32]{{Value: 7}, {Value: 100003}}
	if got := nestedABIArrayRoundTrip(array); got[0].Value != 18 || got[1].Value != 100016 {
		t.Fatalf("aligned array elements corrupted: %+v", got)
	}
}
