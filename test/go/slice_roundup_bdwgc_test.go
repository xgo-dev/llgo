//go:build llgo && !nogc && !baremetal && !wasm

package gotest

import "testing"

func TestSliceRoundupBDWGCBlockSlop(t *testing.T) {
	// Use dynamic lengths so the compiler lowers through MakeSliceRounded
	// rather than constant-size array Alloc.
	for _, tc := range []struct {
		n      int
		minCap int
		maxCap int
	}{
		// 17 bytes fits in BDWGC's 32-byte class (usable <= 31); must not bump to 48.
		{17, 18, 32},
		// 512 bytes lands in BDWGC's 672-byte class with EXTRA_BYTES=1 (usable 671).
		{512, 512, 1024},
		// 2049 bytes fits in a single 4 KiB hblk (usable <= 4095); must not bump to 8 KiB.
		{2049, 2050, 4096},
	} {
		n := tc.n
		s := append([]byte(nil), make([]byte, n)...)
		if cap(s) < tc.minCap || cap(s) > tc.maxCap {
			t.Fatalf("n=%d: cap=%d, want [%d, %d]", n, cap(s), tc.minCap, tc.maxCap)
		}
	}
}
