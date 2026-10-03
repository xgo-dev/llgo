//go:build goexperiment.simd && (arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDByteLookup(t *testing.T) {
	var table [16]int8
	for i := range table {
		table[i] = int8(i*7 - 53)
	}
	vector := archsimd.LoadInt8x16Array(&table)
	// Exhaust all 256 index patterns, including signed-negative and >=16 values.
	for base := 0; base < 256; base += 16 {
		var indices, got [16]int8
		for i := range indices {
			indices[i] = int8(base + i)
		}
		vector.LookupOrZero(archsimd.LoadInt8x16Array(&indices)).StoreArray(&got)
		for i, index := range indices {
			var want int8
			if index >= 0 && index < 16 {
				want = table[index]
			}
			if got[i] != want {
				t.Fatalf("index=%d got=%d want=%d", index, got[i], want)
			}
		}
	}
}

func lookupHexDigits(table, indices archsimd.Uint8x16) archsimd.Uint8x16 {
	return table.BitsToInt8().LookupOrZero(indices.BitsToInt8()).ToBits()
}
