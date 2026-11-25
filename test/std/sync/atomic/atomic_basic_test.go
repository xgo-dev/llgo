package atomic_test

import (
	"sync/atomic"
	"testing"
)

func TestAddInt32(t *testing.T) {
	var i int32 = 10

	// Test positive addition
	result := atomic.AddInt32(&i, 5)
	if result != 15 {
		t.Fatalf("AddInt32(10, 5) = %d, want 15", result)
	}
	if i != 15 {
		t.Fatalf("After AddInt32, i = %d, want 15", i)
	}

	// Test negative addition
	result = atomic.AddInt32(&i, -10)
	if result != 5 {
		t.Fatalf("AddInt32(15, -10) = %d, want 5", result)
	}
	if i != 5 {
		t.Fatalf("After AddInt32, i = %d, want 5", i)
	}

	// Test zero addition
	result = atomic.AddInt32(&i, 0)
	if result != 5 {
		t.Fatalf("AddInt32(5, 0) = %d, want 5", result)
	}
}

func TestAddInt64(t *testing.T) {
	var i int64 = 100

	// Test positive addition
	result := atomic.AddInt64(&i, 50)
	if result != 150 {
		t.Fatalf("AddInt64(100, 50) = %d, want 150", result)
	}
	if i != 150 {
		t.Fatalf("After AddInt64, i = %d, want 150", i)
	}

	// Test negative addition
	result = atomic.AddInt64(&i, -200)
	if result != -50 {
		t.Fatalf("AddInt64(150, -200) = %d, want -50", result)
	}
	if i != -50 {
		t.Fatalf("After AddInt64, i = %d, want -50", i)
	}
}

func TestAddUint32(t *testing.T) {
	var i uint32 = 10

	// Test positive addition
	result := atomic.AddUint32(&i, 5)
	if result != 15 {
		t.Fatalf("AddUint32(10, 5) = %d, want 15", result)
	}
	if i != 15 {
		t.Fatalf("After AddUint32, i = %d, want 15", i)
	}

	// Test addition that causes overflow
	i = ^uint32(0) - 5 // Max value minus 5
	result = atomic.AddUint32(&i, 10)
	// Should wrap around
	if result != 4 {
		t.Fatalf("AddUint32(max-5, 10) = %d, want 4 (wraparound)", result)
	}
}

// Keep the atomic operation in a named function for the linked-code LTO test.
//
//go:noinline
func addUint64(p *uint64, delta uint64) uint64 {
	return atomic.AddUint64(p, delta)
}

func TestAddUint64(t *testing.T) {
	var i uint64 = 10

	// Test positive addition
	result := addUint64(&i, 5)
	if result != 15 {
		t.Fatalf("AddUint64(10, 5) = %d, want 15", result)
	}
	if i != 15 {
		t.Fatalf("After AddUint64, i = %d, want 15", i)
	}

	// Test addition that causes overflow
	i = ^uint64(0) - 5 // Max value minus 5
	result = addUint64(&i, 10)
	// Should wrap around
	if result != 4 {
		t.Fatalf("AddUint64(max-5, 10) = %d, want 4 (wraparound)", result)
	}
}

func TestAddUintptr(t *testing.T) {
	var i uintptr = 10

	// Test positive addition
	result := atomic.AddUintptr(&i, 5)
	if result != 15 {
		t.Fatalf("AddUintptr(10, 5) = %d, want 15", result)
	}
	if i != 15 {
		t.Fatalf("After AddUintptr, i = %d, want 15", i)
	}
}

func TestLoadInt32(t *testing.T) {
	var i int32 = 42

	// Test loading
	result := atomic.LoadInt32(&i)
	if result != 42 {
		t.Fatalf("LoadInt32(42) = %d, want 42", result)
	}
}

func TestLoadInt64(t *testing.T) {
	var i int64 = 42

	// Test loading
	result := atomic.LoadInt64(&i)
	if result != 42 {
		t.Fatalf("LoadInt64(42) = %d, want 42", result)
	}
}

func TestLoadUint32(t *testing.T) {
	var i uint32 = 42

	// Test loading
	result := atomic.LoadUint32(&i)
	if result != 42 {
		t.Fatalf("LoadUint32(42) = %d, want 42", result)
	}
}

func TestLoadUint64(t *testing.T) {
	var i uint64 = 42

	// Test loading
	result := atomic.LoadUint64(&i)
	if result != 42 {
		t.Fatalf("LoadUint64(42) = %d, want 42", result)
	}
}

func TestLoadUintptr(t *testing.T) {
	var i uintptr = 42

	// Test loading
	result := atomic.LoadUintptr(&i)
	if result != 42 {
		t.Fatalf("LoadUintptr(42) = %d, want 42", result)
	}
}

func TestStoreInt32(t *testing.T) {
	var i int32

	// Test storing
	atomic.StoreInt32(&i, 42)
	if i != 42 {
		t.Fatalf("After StoreInt32(42), i = %d, want 42", i)
	}
}

func TestStoreInt64(t *testing.T) {
	var i int64

	// Test storing
	atomic.StoreInt64(&i, 42)
	if i != 42 {
		t.Fatalf("After StoreInt64(42), i = %d, want 42", i)
	}
}

func TestStoreUint32(t *testing.T) {
	var i uint32

	// Test storing
	atomic.StoreUint32(&i, 42)
	if i != 42 {
		t.Fatalf("After StoreUint32(42), i = %d, want 42", i)
	}
}

func TestStoreUint64(t *testing.T) {
	var i uint64

	// Test storing
	atomic.StoreUint64(&i, 42)
	if i != 42 {
		t.Fatalf("After StoreUint64(42), i = %d, want 42", i)
	}
}

func TestStoreUintptr(t *testing.T) {
	var i uintptr

	// Test storing
	atomic.StoreUintptr(&i, 42)
	if i != 42 {
		t.Fatalf("After StoreUintptr(42), i = %d, want 42", i)
	}
}

func TestSwapInt32(t *testing.T) {
	var i int32 = 10

	// Test swapping
	old := atomic.SwapInt32(&i, 20)
	if old != 10 {
		t.Fatalf("SwapInt32(10, 20) = %d, want 10", old)
	}
	if i != 20 {
		t.Fatalf("After SwapInt32, i = %d, want 20", i)
	}
}

func TestSwapInt64(t *testing.T) {
	var i int64 = 10

	// Test swapping
	old := atomic.SwapInt64(&i, 20)
	if old != 10 {
		t.Fatalf("SwapInt64(10, 20) = %d, want 10", old)
	}
	if i != 20 {
		t.Fatalf("After SwapInt64, i = %d, want 20", i)
	}
}

func TestSwapUint32(t *testing.T) {
	var i uint32 = 10

	// Test swapping
	old := atomic.SwapUint32(&i, 20)
	if old != 10 {
		t.Fatalf("SwapUint32(10, 20) = %d, want 10", old)
	}
	if i != 20 {
		t.Fatalf("After SwapUint32, i = %d, want 20", i)
	}
}

func TestSwapUint64(t *testing.T) {
	var i uint64 = 10

	// Test swapping
	old := atomic.SwapUint64(&i, 20)
	if old != 10 {
		t.Fatalf("SwapUint64(10, 20) = %d, want 10", old)
	}
	if i != 20 {
		t.Fatalf("After SwapUint64, i = %d, want 20", i)
	}
}

func TestSwapUintptr(t *testing.T) {
	var i uintptr = 10

	// Test swapping
	old := atomic.SwapUintptr(&i, 20)
	if old != 10 {
		t.Fatalf("SwapUintptr(10, 20) = %d, want 10", old)
	}
	if i != 20 {
		t.Fatalf("After SwapUintptr, i = %d, want 20", i)
	}
}
