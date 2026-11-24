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
}
