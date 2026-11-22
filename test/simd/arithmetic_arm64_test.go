//go:build goexperiment.simd && arm64

package simd_test

import (
	"math/bits"
	"simd/archsimd"
	"testing"
)

func TestSIMDOrNot(t *testing.T) {
	x, y := [4]int32{0, -1, 0x12345678, -17}, [4]int32{-1, 0, -17, 0x12345678}
	var got [4]int32
	archsimd.LoadInt32x4Array(&x).OrNot(archsimd.LoadInt32x4Array(&y)).StoreArray(&got)
	for i := range x {
		if got[i] != x[i]|^y[i] {
			t.Fatalf("lane %d = %x", i, got[i])
		}
	}
}

func TestSIMDNEONIntegerEdges(t *testing.T) {
	in := [16]int8{-128, -127, -65, -3, -2, -1, 0, 1, 2, 3, 7, 15, 63, 64, 126, 127}
	shifts := [16]int8{8, -8, 3, 1, -2, 9, 7, -1, 6, 2, 1, -4, 2, -8, 0, 1}
	x, y := archsimd.LoadInt8x16Array(&in), archsimd.LoadInt8x16Array(&shifts)
	var sat, ones, product [16]int8
	x.ShiftSaturated(y).StoreArray(&sat)
	x.OnesCount().StoreArray(&ones)
	x.MulAdd(y, x).StoreArray(&product)
	var sum int8
	for i, v := range in {
		sum += v
		n := shifts[i]
		w := int(v)
		if n >= 0 {
			w = max(-128, min(127, w<<uint(n)))
		} else {
			w >>= uint(-n)
		}
		if sat[i] != int8(w) || ones[i] != int8(bits.OnesCount8(uint8(v))) || product[i] != v*shifts[i]+v {
			t.Fatalf("lane %d: saturated %d, ones %d, product %d", i, sat[i], ones[i], product[i])
		}
	}
	if x.ReduceSum() != sum || x.ReduceMin() != -128 || x.ReduceMax() != 127 {
		t.Fatal("incorrect signed reductions")
	}
}
