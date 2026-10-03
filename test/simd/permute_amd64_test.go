//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDBytePermuteOrZero(t *testing.T) {
	var table [16]uint8
	for i := range table {
		table[i] = uint8(i*7 + 13)
	}
	vector := archsimd.LoadUint8x16Array(&table)
	for base := 0; base < 256; base += 16 {
		var indices [16]int8
		var got [16]uint8
		for i := range indices {
			indices[i] = int8(base + i)
		}
		vector.PermuteOrZero(archsimd.LoadInt8x16Array(&indices)).StoreArray(&got)
		for i, index := range indices {
			var want uint8
			if index >= 0 {
				want = table[index%16]
			}
			if got[i] != want {
				t.Fatalf("index=%d got=%d want=%d", index, got[i], want)
			}
		}
	}
}

func lookupHexDigits(table, indices archsimd.Uint8x16) archsimd.Uint8x16 {
	return table.PermuteOrZero(indices.BitsToInt8())
}
