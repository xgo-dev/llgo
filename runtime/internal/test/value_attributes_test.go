package test

import "testing"

//go:noinline
//llgo:result nonnull
func checkedValueAttribute[T any](p *T) *T {
	if p == nil {
		panic("nil value attribute")
	}
	return p
}

//go:noinline
//llgo:result range(-4,64) nonnegative
func boundedValueAttribute(n int32) int32 { return n & 63 }

func TestValueAttributes(t *testing.T) {
	x := 42
	if checkedValueAttribute(&x) != &x {
		t.Fatal("pointer result")
	}
	for n := int32(-70); n < 70; n++ {
		if got := boundedValueAttribute(n); got != n&63 {
			t.Fatal(got, n)
		}
	}
	defer func() {
		if recover() != "nil value attribute" {
			t.Fatal("nil panic was lost")
		}
	}()
	checkedValueAttribute[int](nil)
	t.Fatal("nil pointer returned normally")
}
