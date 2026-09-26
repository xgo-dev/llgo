package sync_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

const poolGCMarker = uint64(0x5a17c0de)

//go:noinline
func poolValueRoundTrip(t *testing.T, pool *sync.Pool, collections int) {
	t.Helper()
	value := pool.Get()
	item, ok := value.(*[2]uint64)
	if !ok || item[0] != poolGCMarker {
		t.Fatalf("Pool.Get after %d collections = %T %v", collections, value, value)
	}
	pool.Put(item)
}

func TestPoolAfterGC(t *testing.T) {
	pool := sync.Pool{New: func() any { return &[2]uint64{poolGCMarker, 1} }}
	for i, want := range []string{"0", "1", "2", "3"} {
		poolValueRoundTrip(t, &pool, i)
		if got := fmt.Sprintf("%d", i); got != want {
			t.Fatalf("fmt.Sprintf = %q, want %q", got, want)
		}
		runtime.GC()
		pressure := make([][]byte, 128)
		for j := range pressure {
			pressure[j] = make([]byte, 128)
		}
		runtime.KeepAlive(pressure)
	}
}

func TestPoolBasic(t *testing.T) {
	var pool sync.Pool

	// Test Put and Get
	pool.Put("test")
	if value := pool.Get(); value != "test" {
		t.Fatalf("Pool.Get() returned %v, want 'test'", value)
	}

	// Test Get from empty pool
	if value := pool.Get(); value != nil {
		t.Fatalf("Pool.Get() on empty pool returned %v, want nil", value)
	}
}

func TestPoolMultipleTypes(t *testing.T) {
	var pool sync.Pool

	// Put different types
	pool.Put("string")
	pool.Put(42)
	pool.Put([]int{1, 2, 3})

	// Get values (order not guaranteed)
	values := make([]interface{}, 0, 3)
	for i := 0; i < 3; i++ {
		if v := pool.Get(); v != nil {
			values = append(values, v)
		}
	}

	if len(values) != 3 {
		t.Fatalf("Expected to get 3 values, got %d", len(values))
	}

	// Check that we got the right types
	foundString, foundInt, foundSlice := false, false, false
	for _, v := range values {
		switch val := v.(type) {
		case string:
			if val == "string" {
				foundString = true
			}
		case int:
			if val == 42 {
				foundInt = true
			}
		case []int:
			if len(val) == 3 && val[0] == 1 && val[1] == 2 && val[2] == 3 {
				foundSlice = true
			}
		}
	}

	if !foundString || !foundInt || !foundSlice {
		t.Fatalf("Expected to find all types, got: string=%v, int=%v, slice=%v", foundString, foundInt, foundSlice)
	}
}

func TestPoolNil(t *testing.T) {
	var pool sync.Pool

	// Test Get from empty pool
	if value := pool.Get(); value != nil {
		t.Fatalf("Pool.Get() on new pool should return nil, got %v", value)
	}

	// Put nil and get it back
	pool.Put(nil)
	if value := pool.Get(); value != nil {
		t.Fatalf("Pool.Get() after putting nil should return nil, got %v", value)
	}
}
