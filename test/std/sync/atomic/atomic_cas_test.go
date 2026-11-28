package atomic_test

import (
	"sync/atomic"
	"testing"
)

func TestCompareAndSwapInt32(t *testing.T) {
	var i int32 = 10

	// Test successful compare and swap
	swapped := atomic.CompareAndSwapInt32(&i, 10, 20)
	if !swapped {
		t.Fatalf("CompareAndSwapInt32(10, 10, 20) = false, want true")
	}
	if i != 20 {
		t.Fatalf("After successful CAS, i = %d, want 20", i)
	}

	// Test unsuccessful compare and swap
	swapped = atomic.CompareAndSwapInt32(&i, 10, 30)
	if swapped {
		t.Fatalf("CompareAndSwapInt32(20, 10, 30) = true, want false")
	}
	if i != 20 {
		t.Fatalf("After unsuccessful CAS, i = %d, want 20", i)
	}
}

func TestCompareAndSwapInt64(t *testing.T) {
	var i int64 = 100

	// Test successful compare and swap
	swapped := atomic.CompareAndSwapInt64(&i, 100, 200)
	if !swapped {
		t.Fatalf("CompareAndSwapInt64(100, 100, 200) = false, want true")
	}
	if i != 200 {
		t.Fatalf("After successful CAS, i = %d, want 200", i)
	}

	// Test unsuccessful compare and swap
	swapped = atomic.CompareAndSwapInt64(&i, 100, 300)
	if swapped {
		t.Fatalf("CompareAndSwapInt64(200, 100, 300) = true, want false")
	}
	if i != 200 {
		t.Fatalf("After unsuccessful CAS, i = %d, want 200", i)
	}
}

func TestCompareAndSwapUint32(t *testing.T) {
	var i uint32 = 10

	// Test successful compare and swap
	swapped := atomic.CompareAndSwapUint32(&i, 10, 20)
	if !swapped {
		t.Fatalf("CompareAndSwapUint32(10, 10, 20) = false, want true")
	}
	if i != 20 {
		t.Fatalf("After successful CAS, i = %d, want 20", i)
	}

	// Test unsuccessful compare and swap
	swapped = atomic.CompareAndSwapUint32(&i, 10, 30)
	if swapped {
		t.Fatalf("CompareAndSwapUint32(20, 10, 30) = true, want false")
	}
	if i != 20 {
		t.Fatalf("After unsuccessful CAS, i = %d, want 20", i)
	}
}

// Keep the atomic operation in a named function for the linked-code LTO test.
//
//go:noinline
func compareAndSwapUint64(p *uint64, old, next uint64) bool {
	return atomic.CompareAndSwapUint64(p, old, next)
}

func TestCompareAndSwapUint64(t *testing.T) {
	var i uint64 = 100

	// Test successful compare and swap
	swapped := compareAndSwapUint64(&i, 100, 200)
	if !swapped {
		t.Fatalf("CompareAndSwapUint64(100, 100, 200) = false, want true")
	}
	if i != 200 {
		t.Fatalf("After successful CAS, i = %d, want 200", i)
	}

	// Test unsuccessful compare and swap
	swapped = compareAndSwapUint64(&i, 100, 300)
	if swapped {
		t.Fatalf("CompareAndSwapUint64(200, 100, 300) = true, want false")
	}
	if i != 200 {
		t.Fatalf("After unsuccessful CAS, i = %d, want 200", i)
	}
}

func TestCompareAndSwapUintptr(t *testing.T) {
	var i uintptr = 10

	// Test successful compare and swap
	swapped := atomic.CompareAndSwapUintptr(&i, 10, 20)
	if !swapped {
		t.Fatalf("CompareAndSwapUintptr(10, 10, 20) = false, want true")
	}
	if i != 20 {
		t.Fatalf("After successful CAS, i = %d, want 20", i)
	}

	// Test unsuccessful compare and swap
	swapped = atomic.CompareAndSwapUintptr(&i, 10, 30)
	if swapped {
		t.Fatalf("CompareAndSwapUintptr(20, 10, 30) = true, want false")
	}
	if i != 20 {
		t.Fatalf("After unsuccessful CAS, i = %d, want 20", i)
	}
}
