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
