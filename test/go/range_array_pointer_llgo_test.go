//go:build llgo

package gotest

import "testing"

func TestRangeOverNilArrayPointerCallPanics(t *testing.T) {
	calls := 0
	next := func() *[3]int {
		calls++
		return nil
	}

	expectRangeArrayPanic(t, func() {
		for range *next() {
		}
	})
	if calls != 1 {
		t.Fatalf("range expression calls = %d, want 1", calls)
	}
}

func TestLenOfNilArrayPointerCallPanics(t *testing.T) {
	expectRangeArrayPanic(t, func() {
		_ = len(*nilRangeArrayPointer())
	})
}

func expectRangeArrayPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil array pointer panic")
		}
	}()
	f()
}
