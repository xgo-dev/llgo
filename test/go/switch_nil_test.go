package gotest

import "testing"

func TestSwitchNilCases(t *testing.T) {
	switch f := func() {}; f {
	case nil:
		t.Fatal("non-nil func matched nil")
	default:
	}

	var nilFunc func()
	switch nilFunc {
	case nil:
	default:
		t.Fatal("nil func did not match nil")
	}

	switch m := make(map[int]int); m {
	case nil:
		t.Fatal("non-nil map matched nil")
	default:
	}

	var nilMap map[int]int
	switch nilMap {
	case nil:
	default:
		t.Fatal("nil map did not match nil")
	}

	switch s := make([]int, 1); s {
	case nil:
		t.Fatal("non-nil slice matched nil")
	default:
	}

	var nilSlice []int
	switch nilSlice {
	case nil:
	default:
		t.Fatal("nil slice did not match nil")
	}

	switch c1, c2 := make(chan int), make(chan int); c1 {
	case nil:
		t.Fatal("non-nil channel matched nil")
	case c2:
		t.Fatal("channel matched a different channel")
	case c1:
	default:
		t.Fatal("channel did not match itself")
	}

	switch (*int)(nil) {
	case nil:
	case any(nil):
		t.Fatal("typed nil pointer matched nil interface")
	default:
		t.Fatal("typed nil pointer did not match nil")
	}
}

func TestGenericSwitchNilCases(t *testing.T) {
	checkGenericNilSwitch[int](t)
	checkGenericNilSwitch[string](t)
}

func checkGenericNilSwitch[T any](t *testing.T) {
	switch []T(nil) {
	case nil:
	default:
		t.Fatal("nil generic slice did not match nil")
	}
	if []T(nil) != nil {
		t.Fatal("nil generic slice compared non-nil")
	}

	switch make([]T, 1) {
	case nil:
		t.Fatal("non-nil generic slice matched nil")
	default:
	}

	switch (func() T)(nil) {
	case nil:
	default:
		t.Fatal("nil generic func did not match nil")
	}
	if (func() T)(nil) != nil {
		t.Fatal("nil generic func compared non-nil")
	}

	var zero T
	switch func() T { return zero } {
	case nil:
		t.Fatal("non-nil generic func matched nil")
	default:
	}

	switch (map[int]T)(nil) {
	case nil:
	default:
		t.Fatal("nil generic map did not match nil")
	}
	if (map[int]T)(nil) != nil {
		t.Fatal("nil generic map compared non-nil")
	}

	switch make(map[int]T) {
	case nil:
		t.Fatal("non-nil generic map matched nil")
	default:
	}
}
