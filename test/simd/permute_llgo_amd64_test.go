//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// These official operations require AVX512. Exercise LLGo's baseline-safe
// equivalent against scalar references on machines without that extension.
func TestSIMDModuloPermute(t *testing.T) {
	var table [16]uint8
	for i := range table {
		table[i] = uint8(i*7 + 13)
	}
	vector := archsimd.LoadUint8x16Array(&table)
	for base := 0; base < 256; base += 16 {
		var indices, got [16]uint8
		for i := range indices {
			indices[i] = uint8(base + i)
		}
		vector.Permute(archsimd.LoadUint8x16Array(&indices)).StoreArray(&got)
		for i, index := range indices {
			if got[i] != table[index%16] {
				t.Fatalf("index=%d got=%d", index, got[i])
			}
		}
	}
	table16 := [8]uint16{1, 17, 123, 999, 1024, 50000, 60000, 65535}
	for _, indices := range [][8]uint16{{0, 1, 2, 3, 4, 5, 6, 7}, {8, 9, 10, 11, 12, 13, 14, 15}, {1 << 15, 65535, 256, 257, 258, 259, 260, 261}} {
		var got [8]uint16
		archsimd.LoadUint16x8Array(&table16).Permute(archsimd.LoadUint16x8Array(&indices)).StoreArray(&got)
		for i, index := range indices {
			if got[i] != table16[index%8] {
				t.Fatalf("index=%d got=%x", index, got[i])
			}
		}
	}
}
