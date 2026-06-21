package gotest

import "testing"

type zeroSizeCompareField struct {
	n int
	z struct{}
}

//go:noinline
func zeroSizeDerefEqual(p, q *struct{}) bool {
	return *p == *q
}

//go:noinline
func zeroSizeFieldEqual(p, q *zeroSizeCompareField) bool {
	return p.z == q.z
}

//go:noinline
func zeroSizeResultEqual(p, q func() struct{}) bool {
	return p() == q()
}

func zeroSizeResultEqualInline(p, q func() struct{}) bool {
	return p() == q()
}

func TestZeroSizeNilPointerComparisonPanics(t *testing.T) {
	expectZeroSizeComparePanic(t, func() {
		zeroSizeDerefEqual(nil, nil)
	})
	expectZeroSizeComparePanic(t, func() {
		zeroSizeFieldEqual(nil, nil)
	})
}

func TestZeroSizeFunctionResultComparisonCallsFunctions(t *testing.T) {
	n := 0
	inc := func() struct{} {
		n++
		return struct{}{}
	}
	if !zeroSizeResultEqual(inc, inc) {
		t.Fatal("zero-sized results should compare equal")
	}
	if n != 2 {
		t.Fatalf("calls after noinline comparison = %d, want 2", n)
	}
	if !zeroSizeResultEqualInline(inc, inc) {
		t.Fatal("zero-sized results should compare equal")
	}
	if n != 4 {
		t.Fatalf("calls after inline comparison = %d, want 4", n)
	}
}

func expectZeroSizeComparePanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil pointer dereference panic")
		}
	}()
	f()
}
