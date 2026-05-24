//go:build !llgo

package gotest

import "testing"

func TestRangeOverNilArrayPointerCallIsEvaluated(t *testing.T) {
	calls := 0
	next := func() *[3]int {
		calls++
		return nil
	}

	sum := 0
	for i := range *next() {
		sum += i
	}
	if calls != 1 {
		t.Fatalf("range expression calls = %d, want 1", calls)
	}
	if sum != 3 {
		t.Fatalf("range over nil *array call sum = %d, want 3", sum)
	}
}
