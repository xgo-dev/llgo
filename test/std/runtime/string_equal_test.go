package runtime_test

import (
	"testing"
	"unsafe"
)

//go:noinline
func equalStringValues(a, b string) bool { return a == b }

//go:noinline
func unequalStringValues(a, b string) bool { return a != b }

//go:noinline
func equalStringBytes(a []byte, b string) bool { return string(a) == b }

//go:noinline
func unequalStringBytes(a []byte, b string) bool { return string(a) != b }

// Each view is consumed before returning, so the caller may then mutate b.
func checkStringEqual(t *testing.T, a, b []byte, want bool) {
	t.Helper()
	x := unsafe.String(unsafe.SliceData(a), len(a))
	y := unsafe.String(unsafe.SliceData(b), len(b))
	if equalStringValues(x, y) != want || unequalStringValues(x, y) == want ||
		equalStringBytes(a, y) != want || unequalStringBytes(a, y) == want {
		t.Fatalf("comparison of lengths %d/%d, want equal=%v", len(a), len(b), want)
	}
}

func TestStringEqualBlocks(t *testing.T) {
	for _, size := range []int{0, 1, 2, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 1024} {
		for offset := 0; offset < 8; offset++ {
			a := make([]byte, size+8)[offset:][:size]
			b := make([]byte, size+8)[7-offset:][:size]
			for i := range a {
				a[i], b[i] = byte(i*17+3), byte(i*17+3)
			}
			checkStringEqual(t, a, b, true)
			checkStringEqual(t, a, a, true)
			if size == 0 {
				checkStringEqual(t, nil, b, true)
				continue
			}
			checkStringEqual(t, a[:size-1], b, false)
			checkStringEqual(t, a, b[:size-1], false)
			for _, i := range []int{0, size / 2, size - 1} {
				b[i] ^= 1
				checkStringEqual(t, a, b, false)
				b[i] ^= 1
			}
		}
	}
}
