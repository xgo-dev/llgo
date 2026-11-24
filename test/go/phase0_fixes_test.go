package gotest

import "testing"

func TestStructEqualShortCircuit(t *testing.T) {
	type pair struct {
		tag int
		val any
	}
	a := pair{tag: 1, val: []int{1}}
	b := pair{tag: 2, val: []int{2}}
	if a == b {
		t.Fatal("structs with different first field should not be equal")
	}
	if !(a != b) {
		t.Fatal("structs with different first field should be unequal")
	}

	c := pair{tag: 1, val: []int{2}}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic when comparing uncomparable interface payloads after equal first field")
			}
		}()
		_ = a == c
	}()
}

func TestManyConditionalDefers(t *testing.T) {
	var order []int
	record := func(i int) {
		order = append(order, i)
	}
	run := func(cond bool) {
		if cond {
			defer record(1)
		}
		if cond {
			defer record(2)
		}
		if cond {
			defer record(3)
		}
		if cond {
			defer record(4)
		}
		if cond {
			defer record(5)
		}
		if cond {
			defer record(6)
		}
		if cond {
			defer record(7)
		}
		if cond {
			defer record(8)
		}
		if cond {
			defer record(9)
		}
		if cond {
			defer record(10)
		}
		if cond {
			defer record(11)
		}
		if cond {
			defer record(12)
		}
		if cond {
			defer record(13)
		}
		if cond {
			defer record(14)
		}
		if cond {
			defer record(15)
		}
		if cond {
			defer record(16)
		}
		if cond {
			defer record(17)
		}
		if cond {
			defer record(18)
		}
		if cond {
			defer record(19)
		}
		if cond {
			defer record(20)
		}
		if cond {
			defer record(21)
		}
		if cond {
			defer record(22)
		}
		if cond {
			defer record(23)
		}
		if cond {
			defer record(24)
		}
		if cond {
			defer record(25)
		}
		if cond {
			defer record(26)
		}
		if cond {
			defer record(27)
		}
		if cond {
			defer record(28)
		}
		if cond {
			defer record(29)
		}
		if cond {
			defer record(30)
		}
		if cond {
			defer record(31)
		}
		if cond {
			defer record(32)
		}
		if cond {
			defer record(33)
		}
		if cond {
			defer record(34)
		}
		if cond {
			defer record(35)
		}
		if cond {
			defer record(36)
		}
		if cond {
			defer record(37)
		}
		if cond {
			defer record(38)
		}
		if cond {
			defer record(39)
		}
		if cond {
			defer record(40)
		}
		if cond {
			defer record(41)
		}
		if cond {
			defer record(42)
		}
		if cond {
			defer record(43)
		}
		if cond {
			defer record(44)
		}
		if cond {
			defer record(45)
		}
		if cond {
			defer record(46)
		}
		if cond {
			defer record(47)
		}
		if cond {
			defer record(48)
		}
		if cond {
			defer record(49)
		}
		if cond {
			defer record(50)
		}
		if cond {
			defer record(51)
		}
		if cond {
			defer record(52)
		}
		if cond {
			defer record(53)
		}
		if cond {
			defer record(54)
		}
		if cond {
			defer record(55)
		}
		if cond {
			defer record(56)
		}
		if cond {
			defer record(57)
		}
		if cond {
			defer record(58)
		}
		if cond {
			defer record(59)
		}
		if cond {
			defer record(60)
		}
		if cond {
			defer record(61)
		}
		if cond {
			defer record(62)
		}
		if cond {
			defer record(63)
		}
		if cond {
			defer record(64)
		}
		if cond {
			defer record(65)
		}
		if cond {
			defer record(66)
		}
		if cond {
			defer record(67)
		}
		if cond {
			defer record(68)
		}
	}
	run(false)
	if len(order) != 0 {
		t.Fatalf("run(false) executed %d defers, want 0", len(order))
	}

	run(true)
	if len(order) != 68 {
		t.Fatalf("got %d defers executed, want 68", len(order))
	}
	for i, got := range order {
		want := 68 - i
		if got != want {
			t.Fatalf("defer[%d] = %d, want %d", i, got, want)
		}
	}

	// Verify alternating conditions skip both bitmask-tracked and overflow-fallback defers.
	order = nil
	runAlt := func(a, b bool) {
		if a {
			defer record(1)
		}
		if b {
			defer record(2)
		}
		if a {
			defer record(3)
		}
		if b {
			defer record(4)
		}
		if a {
			defer record(5)
		}
		if b {
			defer record(6)
		}
		if a {
			defer record(7)
		}
		if b {
			defer record(8)
		}
		if a {
			defer record(9)
		}
		if b {
			defer record(10)
		}
		if a {
			defer record(11)
		}
		if b {
			defer record(12)
		}
		if a {
			defer record(13)
		}
		if b {
			defer record(14)
		}
		if a {
			defer record(15)
		}
		if b {
			defer record(16)
		}
		if a {
			defer record(17)
		}
		if b {
			defer record(18)
		}
		if a {
			defer record(19)
		}
		if b {
			defer record(20)
		}
		if a {
			defer record(21)
		}
		if b {
			defer record(22)
		}
		if a {
			defer record(23)
		}
		if b {
			defer record(24)
		}
		if a {
			defer record(25)
		}
		if b {
			defer record(26)
		}
		if a {
			defer record(27)
		}
		if b {
			defer record(28)
		}
		if a {
			defer record(29)
		}
		if b {
			defer record(30)
		}
		if a {
			defer record(31)
		}
		if b {
			defer record(32)
		}
		if a {
			defer record(33)
		}
		if b {
			defer record(34)
		}
		if a {
			defer record(35)
		}
		if b {
			defer record(36)
		}
		if a {
			defer record(37)
		}
		if b {
			defer record(38)
		}
		if a {
			defer record(39)
		}
		if b {
			defer record(40)
		}
		if a {
			defer record(41)
		}
		if b {
			defer record(42)
		}
		if a {
			defer record(43)
		}
		if b {
			defer record(44)
		}
		if a {
			defer record(45)
		}
		if b {
			defer record(46)
		}
		if a {
			defer record(47)
		}
		if b {
			defer record(48)
		}
		if a {
			defer record(49)
		}
		if b {
			defer record(50)
		}
		if a {
			defer record(51)
		}
		if b {
			defer record(52)
		}
		if a {
			defer record(53)
		}
		if b {
			defer record(54)
		}
		if a {
			defer record(55)
		}
		if b {
			defer record(56)
		}
		if a {
			defer record(57)
		}
		if b {
			defer record(58)
		}
		if a {
			defer record(59)
		}
		if b {
			defer record(60)
		}
		if a {
			defer record(61)
		}
		if b {
			defer record(62)
		}
		if a {
			defer record(63)
		}
		if b {
			defer record(64)
		}
		if a {
			defer record(65)
		}
		if b {
			defer record(66)
		}
		if a {
			defer record(67)
		}
		if b {
			defer record(68)
		}
	}
	runAlt(true, false)
	if len(order) != 34 {
		t.Fatalf("runAlt(true, false) executed %d defers, want 34", len(order))
	}
	for i, got := range order {
		want := 67 - 2*i
		if got != want {
			t.Fatalf("alt defer[%d] = %d, want %d", i, got, want)
		}
	}
}
