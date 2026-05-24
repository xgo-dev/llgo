package gotest

import "testing"

type rangeArrayPointerHolder struct {
	data *[3]int
}

//go:noinline
func nilRangeArrayPointer() *[3]int {
	return nil
}

func TestRangeOverNilArrayPointerUsesLength(t *testing.T) {
	var p *[3]int

	sum := 0
	for i := range *p {
		sum += i
	}
	if sum != 3 {
		t.Fatalf("range over nil *array sum = %d, want 3", sum)
	}
}

func TestRangeOverNilArrayPointerFieldUsesLength(t *testing.T) {
	holder := &rangeArrayPointerHolder{}

	sum := 0
	for i := range *holder.data {
		sum += i
	}
	if sum != 3 {
		t.Fatalf("range over nil *array field sum = %d, want 3", sum)
	}
}

func TestLenOfNilArrayPointerValueUsesStaticLength(t *testing.T) {
	var p *[3]int
	if got := len(*p); got != 3 {
		t.Fatalf("len(*nil array pointer) = %d, want 3", got)
	}
	if got := len(p); got != 3 {
		t.Fatalf("len(nil array pointer) = %d, want 3", got)
	}
	if got := len(nilRangeArrayPointer()); got != 3 {
		t.Fatalf("len(nil array pointer call) = %d, want 3", got)
	}
}
