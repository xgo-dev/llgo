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

func TestPoolManyOwners(t *testing.T) {
	type value struct{ owner int }
	var pools [128]sync.Pool
	for i := range pools {
		owner := i
		pools[i].New = func() any { return &value{owner} }
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for round := 0; round < 8; round++ {
				for index := range pools {
					i := (index + worker) % len(pools)
					v := pools[i].Get().(*value)
					if v.owner != i {
						t.Errorf("pool %d returned owner %d", i, v.owner)
						return
					}
					pools[i].Put(v)
				}
				if worker == 0 {
					runtime.GC()
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
}

func TestPoolBasic(t *testing.T) {
	var pool sync.Pool

	// Test Put and Get
	pool.Put("test")
	if value := pool.Get(); value != nil && value != "test" {
		t.Fatalf("Pool.Get() returned %v, want nil or 'test'", value)
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

	// A pool may discard entries. Check every value it retains.
	foundString, foundInt, foundSlice := false, false, false
	for range 3 {
		v := pool.Get()
		switch val := v.(type) {
		case nil:
		case string:
			if foundString || val != "string" {
				t.Fatalf("unexpected pooled string %q", val)
			}
			foundString = true
		case int:
			if foundInt || val != 42 {
				t.Fatalf("unexpected pooled integer %d", val)
			}
			foundInt = true
		case []int:
			if foundSlice || len(val) != 3 || val[0] != 1 || val[1] != 2 || val[2] != 3 {
				t.Fatalf("unexpected pooled slice %v", val)
			}
			foundSlice = true
		default:
			t.Fatalf("unexpected pooled type %T", v)
		}
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
