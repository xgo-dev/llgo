package bytes_test

import (
	"bytes"
	"strconv"
	"testing"
)

//go:noinline
func equalBytes(a, b []byte) bool { return bytes.Equal(a, b) }

//go:noinline
func equalSnapshot(a, b []byte) bool {
	s := string(a)
	b[0] ^= 1
	return s == string(b)
}

func TestEqualAllocations(t *testing.T) {
	for _, size := range []int{16, 65536} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			a := bytes.Repeat([]byte{3}, size)
			b := bytes.Clone(a)
			if !equalBytes(a, b) || !equalBytes(nil, nil) || !equalBytes(nil, []byte{}) || equalBytes(a, b[:size-1]) {
				t.Fatal("incorrect equality or length comparison")
			}
			b[size-1] ^= 1
			if equalBytes(a, b) {
				t.Fatal("different final byte compared equal")
			}
			b[size-1] ^= 1
			if !equalBytes(a[:size-1], a[1:]) {
				t.Fatal("overlapping equal slices compared unequal")
			}
			if got := testing.AllocsPerRun(100, func() {
				if !equalBytes(a, b) {
					t.Fatal("equal slices compared unequal")
				}
			}); got != 0 {
				t.Fatalf("Equal allocated %v objects per call", got)
			}
			if equalSnapshot(a, a) {
				t.Fatal("conversion lost the snapshot before an aliased write")
			}
		})
	}
}

func BenchmarkEqualBytes(b *testing.B) {
	for _, size := range []int{16, 65536} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			a := bytes.Repeat([]byte{3}, size)
			other := bytes.Clone(a)
			b.SetBytes(int64(size))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if !equalBytes(a, other) {
					b.Fatal("equal slices compared unequal")
				}
			}
		})
	}
}
