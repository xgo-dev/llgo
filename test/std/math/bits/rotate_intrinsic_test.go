package bits_test

import (
	"math/bits"
	"testing"
)

// Keep the calls indirect even with optimization, exercising the ordinary Go
// wrapper's function-value ABI rather than just direct intrinsic lowering.
//
//go:noinline
func checkRotate8(t *testing.T, rotate func(uint8, int) uint8) {
	for x := 0; x < 256; x++ {
		for k := -257; k <= 257; k++ {
			s := uint(k) & 7
			want := uint8(x)<<s | uint8(x)>>(8-s)
			if got := rotate(uint8(x), k); got != want {
				t.Fatalf("RotateLeft8(%x, %d) = %x, want %x", x, k, got, want)
			}
		}
	}
}

func TestRotateIntrinsicEdges(t *testing.T) {
	checkRotate8(t, bits.RotateLeft8)
	maxInt := int(^uint(0) >> 1)
	counts := []int{-maxInt - 1, -maxInt, -257, -65, -64, -33, -32, -17, -16, -9, -8, -1, 0, 1, 7, 8, 15, 16, 31, 32, 63, 64, 65, 257, maxInt}
	for _, x := range []uint64{0, 1, ^uint64(0), 0x8000000000000001, 0x9182736455aa00ff} {
		for _, k := range counts {
			s16, s32, s64, sw := uint(k)&15, uint(k)&31, uint(k)&63, uint(k)&uint(bits.UintSize-1)
			if got, want := bits.RotateLeft16(uint16(x), k), uint16(x)<<s16|uint16(x)>>(16-s16); got != want {
				t.Fatalf("RotateLeft16(%x, %d) = %x, want %x", x, k, got, want)
			}
			if got, want := bits.RotateLeft32(uint32(x), k), uint32(x)<<s32|uint32(x)>>(32-s32); got != want {
				t.Fatalf("RotateLeft32(%x, %d) = %x, want %x", x, k, got, want)
			}
			if got, want := bits.RotateLeft64(x, k), x<<s64|x>>(64-s64); got != want {
				t.Fatalf("RotateLeft64(%x, %d) = %x, want %x", x, k, got, want)
			}
			if got, want := bits.RotateLeft(uint(x), k), uint(x)<<sw|uint(x)>>(uint(bits.UintSize)-sw); got != want {
				t.Fatalf("RotateLeft(%x, %d) = %x, want %x", x, k, got, want)
			}
		}
	}
}
