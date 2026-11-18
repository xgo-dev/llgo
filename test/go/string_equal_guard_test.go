//go:build linux || darwin || windows

package gotest

import (
	"testing"
	"unsafe"
)

//go:noinline
func guardEqualStrings(a, b string) bool { return a == b }

//go:noinline
func guardEqualStringBytes(a []byte, b string) bool { return string(a) == b }

func TestStringEqualPageBoundary(t *testing.T) {
	left, pageSize := protectedMemory(t, 2, 1, 1)
	right, _ := protectedMemory(t, 2, 1, 1)
	for _, n := range []int{0, 1, 2, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 1024} {
		a, b := left[pageSize-n:pageSize], right[pageSize-n:pageSize]
		for i := range a {
			a[i], b[i] = byte(i*17+3), byte(i*17+3)
		}
		x := unsafe.String((*byte)(unsafe.Add(unsafe.Pointer(&left[0]), pageSize-n)), n)
		y := unsafe.String((*byte)(unsafe.Add(unsafe.Pointer(&right[0]), pageSize-n)), n)
		if !guardEqualStrings(x, y) || !guardEqualStringBytes(a, y) {
			t.Fatalf("equal boundary strings, length %d", n)
		}
		if n > 0 {
			// Stop using the earlier string views before mutating their bytes.
			x, y = "", ""
			b[n-1] ^= 1
			x = unsafe.String(unsafe.SliceData(a), n)
			y = unsafe.String(unsafe.SliceData(b), n)
			if guardEqualStrings(x, y) || guardEqualStringBytes(a, y) {
				t.Fatalf("unequal boundary strings, length %d", n)
			}
		}
	}
	// These addresses point into inaccessible pages. Length and pointer checks
	// must return without reading data, including non-nil empty strings.
	p := (*byte)(unsafe.Add(unsafe.Pointer(&left[0]), pageSize))
	q := (*byte)(unsafe.Add(unsafe.Pointer(&right[0]), pageSize))
	if !guardEqualStrings(unsafe.String(p, 0), unsafe.String(q, 0)) ||
		guardEqualStrings(unsafe.String(p, 1), "") ||
		!guardEqualStrings(unsafe.String(p, 1), unsafe.String(p, 1)) {
		t.Fatal("string equality read data before checking lengths and pointers")
	}
}
