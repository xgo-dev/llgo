package atomic_test

import (
	"sync/atomic"
	"testing"
)

func TestAndInt32(t *testing.T) {
	var i int32 = 0xFFAA // 1111 1111 1010 1010

	// Test AND operation
	oldVal := atomic.AndInt32(&i, 0xF0F0) // 1111 0000 1111 0000
	if oldVal != 0xFFAA {
		t.Fatalf("AndInt32(0xFFAA, 0xF0F0) returned old value = %x, want 0xFFAA", oldVal)
	}
	if i != 0xF0A0 { // FFAA & F0F0 = F0A0
		t.Fatalf("After AndInt32, i = %x, want 0xF0A0", i)
	}

	// Test with zero
	oldVal = atomic.AndInt32(&i, 0)
	if oldVal != 0xF0A0 {
		t.Fatalf("AndInt32(0xF0A0, 0) returned old value = %x, want 0xF0A0", oldVal)
	}
	if i != 0 {
		t.Fatalf("After AndInt32 with zero, i = %x, want 0", i)
	}
}

func TestAndInt64(t *testing.T) {
	var i int64 = 0xFFAAFFAA
	oldVal := atomic.AndInt64(&i, 0xF0F0F0F0)
	if oldVal != 0xFFAAFFAA {
		t.Fatalf("AndInt64 returned old value = %x, want 0xFFAAFFAA", oldVal)
	}
	if i != 0xF0A0F0A0 {
		t.Fatalf("After AndInt64, i = %x, want 0xF0A0F0A0", i)
	}
}

func TestAndUint32(t *testing.T) {
	var i uint32 = 0xFFAA
	oldVal := atomic.AndUint32(&i, 0xF0F0)
	if oldVal != 0xFFAA {
		t.Fatalf("AndUint32 returned old value = %x, want 0xFFAA", oldVal)
	}
	if i != 0xF0A0 {
		t.Fatalf("After AndUint32, i = %x, want 0xF0A0", i)
	}
}

func TestAndUint64(t *testing.T) {
	var i uint64 = 0xFFAAFFAAFFAAFFAA
	oldVal := atomic.AndUint64(&i, 0xF0F0F0F0F0F0F0F0)
	if oldVal != 0xFFAAFFAAFFAAFFAA {
		t.Fatalf("AndUint64 returned old value = %x, want 0xFFAAFFAAFFAAFFAA", oldVal)
	}
	if i != 0xF0A0F0A0F0A0F0A0 {
		t.Fatalf("After AndUint64, i = %x, want 0xF0A0F0A0F0A0F0A0", i)
	}
}

func TestAndUintptr(t *testing.T) {
	var i uintptr = 0xFFAA
	oldVal := atomic.AndUintptr(&i, 0xF0F0)
	if oldVal != 0xFFAA {
		t.Fatalf("AndUintptr returned old value = %x, want 0xFFAA", oldVal)
	}
	if i != 0xF0A0 {
		t.Fatalf("After AndUintptr, i = %x, want 0xF0A0", i)
	}
}

func TestOrInt32(t *testing.T) {
	var i int32 = 0x5050 // 0101 0000 0101 0000

	// Test OR operation
	oldVal := atomic.OrInt32(&i, 0x0A0A) // 0000 1010 0000 1010
	if oldVal != 0x5050 {
		t.Fatalf("OrInt32(0x5050, 0x0A0A) returned old value = %x, want 0x5050", oldVal)
	}
	if i != 0x5A5A {
		t.Fatalf("After OrInt32, i = %x, want 0x5A5A", i)
	}

	// Test with zero
	oldVal = atomic.OrInt32(&i, 0)
	if oldVal != 0x5A5A {
		t.Fatalf("OrInt32(0x5A5A, 0) returned old value = %x, want 0x5A5A", oldVal)
	}
	if i != 0x5A5A {
		t.Fatalf("After OrInt32 with zero, i = %x, want 0x5A5A", i)
	}
}

func TestOrInt64(t *testing.T) {
	var i int64 = 0x5050505050505050
	oldVal := atomic.OrInt64(&i, 0x0A0A0A0A0A0A0A0A)
	if oldVal != 0x5050505050505050 {
		t.Fatalf("OrInt64 returned old value = %x, want 0x5050505050505050", oldVal)
	}
	if i != 0x5A5A5A5A5A5A5A5A {
		t.Fatalf("After OrInt64, i = %x, want 0x5A5A5A5A5A5A5A5A", i)
	}
}

func TestOrUint32(t *testing.T) {
	var i uint32 = 0x5050
	oldVal := atomic.OrUint32(&i, 0x0A0A)
	if oldVal != 0x5050 {
		t.Fatalf("OrUint32 returned old value = %x, want 0x5050", oldVal)
	}
	if i != 0x5A5A {
		t.Fatalf("After OrUint32, i = %x, want 0x5A5A", i)
	}
}

func TestOrUint64(t *testing.T) {
	var i uint64 = 0x5050505050505050
	oldVal := atomic.OrUint64(&i, 0x0A0A0A0A0A0A0A0A)
	if oldVal != 0x5050505050505050 {
		t.Fatalf("OrUint64 returned old value = %x, want 0x5050505050505050", oldVal)
	}
	if i != 0x5A5A5A5A5A5A5A5A {
		t.Fatalf("After OrUint64, i = %x, want 0x5A5A5A5A5A5A5A5A", i)
	}
}

func TestOrUintptr(t *testing.T) {
	var i uintptr = 0x5050
	oldVal := atomic.OrUintptr(&i, 0x0A0A)
	if oldVal != 0x5050 {
		t.Fatalf("OrUintptr returned old value = %x, want 0x5050", oldVal)
	}
	if i != 0x5A5A {
		t.Fatalf("After OrUintptr, i = %x, want 0x5A5A", i)
	}
}

// Type methods for bitwise operations are moved to atomic_bitwise_methods_test.go
// to exclude them from LLGo builds due to platform-specific behavior.
