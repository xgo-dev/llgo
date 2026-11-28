package gotest

import (
	"bytes"
	"io"
	"testing"
)

func TestSliceRoundupAndFreshNilAppend(t *testing.T) {
	// 1. Empty fresh nil-append must preserve nil and cap 0.
	n0 := 0
	if s := append([]byte(nil), make([]byte, n0)...); s != nil || len(s) != 0 || cap(s) != 0 {
		t.Fatalf("empty fresh append: got (%v, len=%d, cap=%d), want (nil, 0, 0)", s, len(s), cap(s))
	}

	// 2. Zero-sized elements must fall back cleanly without panicking.
	nz := 17
	if zs := append([]struct{}(nil), make([]struct{}, nz)...); len(zs) != nz || cap(zs) < nz {
		t.Fatalf("zero-sized fresh append: len=%d cap=%d", len(zs), cap(zs))
	}

	// 3. Dynamic fresh nil-append across BDWGC granule/block boundaries:
	// capacity must be >= len, spare capacity must be zeroed, and rounding
	// must never bump a request past its own BDWGC block.
	sizes := []int{1, 8, 15, 16, 17, 31, 32, 33, 384, 385, 511, 512, 513, 1023, 1024, 2047, 2048, 2049, 4095, 4096}
	for _, n := range sizes {
		s := append([]byte(nil), make([]byte, n)...)
		if len(s) != n || cap(s) < n {
			t.Fatalf("n=%d: len=%d cap=%d", n, len(s), cap(s))
		}
		full := s[:cap(s)]
		for i, b := range full {
			if b != 0 {
				t.Fatalf("n=%d: non-zero byte at index %d (cap=%d): %d", n, i, cap(s), b)
			}
		}
		// Ensure writing all bytes up to cap(s) stays inside the allocation.
		full[0] = 0x5a
		if len(full) > 1 {
			full[len(full)-1] = 0xa5
			if full[0] != 0x5a || full[len(full)-1] != 0xa5 {
				t.Fatalf("n=%d: write/read at capacity edges failed", n)
			}
		} else if full[0] != 0x5a {
			t.Fatalf("n=%d: write/read at capacity edge failed", n)
		}
	}

	// 4. Pointer-bearing slices also expose zeroed spare capacity.
	for _, n := range []int{1, 3, 5, 17, 65} {
		ps := append([]*int(nil), make([]*int, n)...)
		if len(ps) != n || cap(ps) < n {
			t.Fatalf("ptr n=%d: len=%d cap=%d", n, len(ps), cap(ps))
		}
		for i, p := range ps[:cap(ps)] {
			if p != nil {
				t.Fatalf("ptr n=%d: non-nil element at %d", n, i)
			}
		}
	}

	// 5. GrowSlice via append and io.ReadAll preserve zeroed spare capacity.
	var g []byte
	for i := 0; i < 5000; i++ {
		g = append(g, byte(i))
	}
	for i := len(g); i < cap(g); i++ {
		if g[:cap(g)][i] != 0 {
			t.Fatalf("GrowSlice left non-zero spare byte at %d", i)
		}
	}

	payload := bytes.Repeat([]byte("abcdefgh"), 1024)
	got, err := io.ReadAll(bytes.NewReader(payload))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("io.ReadAll failed: len=%d err=%v", len(got), err)
	}
	if cap(got) < len(payload) {
		t.Fatalf("io.ReadAll cap=%d < len=%d", cap(got), len(payload))
	}
}

func TestStringToBytesAtomicAlloc(t *testing.T) {
	if b := []byte(""); len(b) != 0 {
		t.Fatalf("empty string to bytes: len=%d", len(b))
	}
	for _, n := range []int{1, 15, 16, 33, 512, 4096} {
		src := string(bytes.Repeat([]byte{'x'}, n))
		b := []byte(src)
		if len(b) != n {
			t.Fatalf("n=%d: len=%d", n, len(b))
		}
		for i, c := range b {
			if c != 'x' {
				t.Fatalf("n=%d: mismatch at %d: %q", n, i, c)
			}
		}
		b[0] = 'y'
		if src[0] != 'x' {
			t.Fatalf("n=%d: mutating []byte(s) mutated source string", n)
		}
	}
}
