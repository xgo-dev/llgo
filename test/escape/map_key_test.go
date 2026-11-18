package escape

import (
	"runtime"
	"testing"
)

//go:noinline
func mapKeyLookup[K comparable](m map[K]int, key K) (int, bool) {
	value, ok := m[key]
	return value, ok
}

//go:noinline
func mapKeyUpdate[K comparable](m map[K]int, key K, value int) {
	m[key] = value
}

//go:noinline
func mapKeyDelete[K comparable](m map[K]int, key K) {
	delete(m, key)
}

func exerciseMapKeys[K comparable](t *testing.T, key func(int) K) {
	t.Helper()
	m := make(map[K]int)
	// Repeated execution reuses each temporary slot, but the map must retain
	// distinct copies, including when its buckets grow or keys are indirect.
	for i := 0; i < 256; i++ {
		mapKeyUpdate(m, key(i), i+1)
		if i%32 == 0 {
			runtime.GC()
		}
	}
	for i := 0; i < 256; i++ {
		if got, ok := mapKeyLookup(m, key(i)); !ok || got != i+1 {
			t.Fatalf("key %d: got %d, %v", i, got, ok)
		}
		if i%2 == 0 {
			mapKeyDelete(m, key(i))
		}
	}
	runtime.GC()
	for i := 0; i < 256; i++ {
		got, ok := mapKeyLookup(m, key(i))
		if ok != (i%2 != 0) || ok && got != i+1 {
			t.Fatalf("key %d after deletion: got %d, %v", i, got, ok)
		}
	}
}

func TestMapKeyTemporaryOwnership(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		type key struct{ A, B int64 }
		exerciseMapKeys(t, func(i int) key { return key{int64(i), int64(-i)} })
	})
	t.Run("indirect", func(t *testing.T) {
		type key [33]uint64
		exerciseMapKeys(t, func(i int) (k key) { k[0], k[32] = uint64(i), uint64(i+1); return })
	})
	t.Run("pointers", func(t *testing.T) {
		type key struct {
			P *int
			N int
		}
		values := make([]int, 256)
		for i := range values {
			values[i] = i
		}
		exerciseMapKeys(t, func(i int) key { return key{&values[i], i} })
	})
	t.Run("interface", func(t *testing.T) {
		type key struct{ Value any }
		exerciseMapKeys(t, func(i int) key { return key{[2]int{i, -i}} })
	})
}

func TestMapKeyTemporaryPanicAndZeroSize(t *testing.T) {
	type key struct{ Value any }
	m := map[key]int{{1}: 42}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("unhashable key did not panic")
			}
		}()
		mapKeyLookup(m, key{[]int{1}})
	}()
	runtime.GC()
	if value, ok := mapKeyLookup(m, key{1}); !ok || value != 42 {
		t.Fatalf("lookup after panic: %d, %v", value, ok)
	}
	empty := make(map[struct{}]int)
	mapKeyUpdate(empty, struct{}{}, 7)
	if value, ok := mapKeyLookup(empty, struct{}{}); !ok || value != 7 {
		t.Fatalf("zero-sized key: %d, %v", value, ok)
	}
	mapKeyDelete(empty, struct{}{})
	if len(empty) != 0 {
		t.Fatal("zero-sized key was not deleted")
	}
}

type ownedMapKey struct {
	Padding [256]byte
	P       *int
	N       int
}

//go:noinline
func makeOwnedMapKeys() map[ownedMapKey]int {
	m := make(map[ownedMapKey]int)
	for i := 0; i < 256; i++ {
		p := new(int)
		*p = i + 1
		mapKeyUpdate(m, ownedMapKey{P: p, N: i}, i+1)
		if i%32 == 0 {
			runtime.GC()
		}
	}
	return m
}

func TestMapKeyPointerContentsSurviveGC(t *testing.T) {
	m := makeOwnedMapKeys()
	// The constructing frame has returned. Only the map owns these indirect,
	// pointer-bearing key copies and their pointees.
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	if len(m) != 256 {
		t.Fatalf("map has %d entries, want 256", len(m))
	}
	for key, value := range m {
		if key.P == nil || *key.P != value || key.N != value-1 {
			t.Fatalf("key contents corrupted for value %d", value)
		}
	}
}
