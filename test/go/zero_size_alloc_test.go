package gotest

import "testing"

var zeroSizeGlobalX, zeroSizeGlobalY [0]int
var zeroSizeHeapP, zeroSizeHeapQ = new([0]int), new([0]int)
var zeroSizeStringArray [0]string
var zeroSizeEmptyStruct struct{}
var zeroSizeStringArrayP = new([0]string)
var zeroSizeEmptyStructP = new(struct{})

func TestZeroSizedGlobalsAndHeapAllocsShareBase(t *testing.T) {
	// LLGO intentionally gives package globals and heap allocations for
	// zero-sized values a shared base, which is stricter than the Go spec.
	if zeroSizeHeapP != zeroSizeHeapQ {
		t.Fatalf("new([0]int) returned different addresses: %p != %p", zeroSizeHeapP, zeroSizeHeapQ)
	}
	if &zeroSizeGlobalX != zeroSizeHeapP {
		t.Fatalf("&global zero-sized array = %p, want heap zero base %p", &zeroSizeGlobalX, zeroSizeHeapP)
	}
	if &zeroSizeGlobalY != zeroSizeHeapP {
		t.Fatalf("&second global zero-sized array = %p, want heap zero base %p", &zeroSizeGlobalY, zeroSizeHeapP)
	}
	if &zeroSizeStringArray != zeroSizeStringArrayP {
		t.Fatalf("&zero-sized string array = %p, want heap zero base %p", &zeroSizeStringArray, zeroSizeStringArrayP)
	}
	if &zeroSizeEmptyStruct != zeroSizeEmptyStructP {
		t.Fatalf("&empty struct = %p, want heap zero base %p", &zeroSizeEmptyStruct, zeroSizeEmptyStructP)
	}
}
