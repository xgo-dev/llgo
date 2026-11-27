package gotest

import (
	"reflect"
	"testing"
)

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

type smallScalarPair32 struct {
	a, b uint32
}

type smallScalarPair64 struct {
	a, b uint64
}

type smallScalarPadded struct {
	a int16
	b int32
	c bool
}

func testReflectMapOfSmallScalar[K comparable](t *testing.T, keys []K) {
	t.Helper()
	t.Run(reflect.TypeFor[K]().String(), func(t *testing.T) {
		mapTyp := reflect.MapOf(reflect.TypeFor[K](), reflect.TypeFor[int]())
		rv := reflect.MakeMap(mapTyp)
		for i, k := range keys {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(i+10))
		}
		staticMap, ok := rv.Interface().(map[K]int)
		if !ok {
			t.Fatalf("reflect.MakeMap(%v) did not convert to static map", mapTyp)
		}
		if len(staticMap) != len(keys) {
			t.Fatalf("static map len = %d, want %d", len(staticMap), len(keys))
		}
		for i, k := range keys {
			if got, ok := staticMap[k]; !ok || got != i+10 {
				t.Fatalf("static lookup staticMap[%+v] = (%d, %v), want (%d, true)", k, got, ok, i+10)
			}
			staticMap[k] = i + 100
			gotVal := rv.MapIndex(reflect.ValueOf(k))
			if !gotVal.IsValid() || int(gotVal.Int()) != i+100 {
				t.Fatalf("reflect lookup MapIndex(%+v) = %v, want %d", k, gotVal, i+100)
			}
		}
		for _, k := range keys {
			delete(staticMap, k)
			if rv.MapIndex(reflect.ValueOf(k)).IsValid() {
				t.Fatalf("reflect MapIndex(%+v) still valid after static delete", k)
			}
		}
		if rv.Len() != 0 {
			t.Fatalf("reflect map len = %d after deletes, want 0", rv.Len())
		}
	})
}

func TestReflectMapOfSmallScalarStructKeys(t *testing.T) {
	testReflectMapOfSmallScalar(t, []smallScalarPair32{
		{0, 0}, {1, 2}, {^uint32(0), 7}, {42, 99},
	})
	testReflectMapOfSmallScalar(t, []smallScalarPair64{
		{0, 0}, {1, 1 << 40}, {^uint64(0), 3},
	})
	testReflectMapOfSmallScalar(t, []smallScalarPadded{
		{0, 0, false}, {-1, 123456, true}, {7, -99, false}, {7, -99, true},
	})
}
