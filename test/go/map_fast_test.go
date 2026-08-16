package gotest

import "testing"

type mapFastNamed32 uint32
type mapFastNamedString string

type mapFastStructKey struct {
	value uint64
}

func testMapFastPath[K comparable](t *testing.T, keys []K) {
	t.Helper()

	m := make(map[K]int)
	for i, key := range keys {
		m[key] = i + 1
	}
	for i, key := range keys {
		if got, ok := m[key]; !ok || got != i+1 {
			t.Fatalf("m[%v] = (%d, %v), want (%d, true)", key, got, ok, i+1)
		}
	}
	for _, key := range keys {
		delete(m, key)
	}
	if len(m) != 0 {
		t.Fatalf("len(m) = %d after delete, want 0", len(m))
	}
}

func TestMapFastPaths(t *testing.T) {
	a, b := 1, 2
	ch1, ch2 := make(chan int), make(chan int)

	testMapFastPath(t, []int32{-1, 0, 1, 1 << 30})
	testMapFastPath(t, []uint64{0, 1, 1 << 40, ^uint64(0)})
	testMapFastPath(t, []mapFastNamed32{0, 1, 1 << 30})
	testMapFastPath(t, []mapFastNamedString{"", "short", "named string"})
	testMapFastPath(t, []string{"", "short", "a string long enough to use the long-key lookup path"})
	testMapFastPath(t, []*int{nil, &a, &b})
	testMapFastPath(t, []chan int{nil, ch1, ch2})
	testMapFastPath(t, []mapFastStructKey{{0}, {1}, {1 << 40}})
}
