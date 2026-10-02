//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

// Package initialization must be able to call both Go broadcast helpers and
// their bodyless implementation, before any user function runs.
var memoryResult archsimd.Float32x4

var broadcastAtInit = archsimd.BroadcastFloat32x4(3.5)

//go:noinline
func indirectLoad(f func(*[4]float32) archsimd.Float32x4, p *[4]float32) archsimd.Float32x4 {
	return f(p)
}

func TestSIMDMemory(t *testing.T) {
	values := [4]float32{math.Float32frombits(0x80000000), math.Float32frombits(0x7fc12345), 3.25, -7.5}
	for offset := 0; offset < 4; offset++ {
		// Exercise element-aligned addresses at every offset within a vector.
		src := make([]float32, 8)
		dst := make([]float32, 8)
		copy(src[offset:], values[:])
		for i := range dst {
			dst[i] = 123
		}
		v := archsimd.LoadFloat32x4(src[offset:])
		store := v.Store
		store(dst[offset:])
		for i := range dst {
			want := float32(123)
			if i >= offset && i < offset+4 {
				want = values[i-offset]
			}
			if math.Float32bits(dst[i]) != math.Float32bits(want) {
				t.Fatalf("offset %d lane %d: got %08x want %08x", offset, i, math.Float32bits(dst[i]), math.Float32bits(want))
			}
		}
	}
	v := indirectLoad(archsimd.LoadFloat32x4Array, &values)
	var copied [4]float32
	v.StoreArray(&copied)
	for i := range values {
		if math.Float32bits(copied[i]) != math.Float32bits(values[i]) {
			t.Fatalf("indirect load lane %d", i)
		}
	}

	bytes := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 255}
	var byteCopy [16]byte
	archsimd.LoadUint8x16Array(&bytes).StoreArray(&byteCopy)
	if byteCopy != bytes {
		t.Fatal("byte memory roundtrip")
	}
	words := [2]int64{math.MinInt64, math.MaxInt64}
	var wordCopy [2]int64
	archsimd.LoadInt64x2Array(&words).StoreArray(&wordCopy)
	if wordCopy != words {
		t.Fatal("integer memory roundtrip")
	}
	doubles := [2]float64{math.Inf(-1), math.SmallestNonzeroFloat64}
	var doubleCopy [2]float64
	archsimd.LoadFloat64x2Array(&doubles).StoreArray(&doubleCopy)
	if doubleCopy != doubles {
		t.Fatal("double memory roundtrip")
	}
}

func TestSIMDBroadcast(t *testing.T) {
	var floats [4]float32
	broadcastAtInit.StoreArray(&floats)
	if floats != [4]float32{3.5, 3.5, 3.5, 3.5} {
		t.Fatal(floats)
	}
	var bytes [16]int8
	archsimd.BroadcastInt8x16(-37).StoreArray(&bytes)
	for i, x := range bytes {
		if x != -37 {
			t.Fatalf("byte lane %d = %d", i, x)
		}
	}
	var words [2]uint64
	archsimd.BroadcastUint64x2(0xfedcba9876543210).StoreArray(&words)
	if words != [2]uint64{0xfedcba9876543210, 0xfedcba9876543210} {
		t.Fatal(words)
	}
	var doubles [2]float64
	archsimd.BroadcastFloat64x2(math.Copysign(0, -1)).StoreArray(&doubles)
	for _, x := range doubles {
		if math.Float64bits(x) != 1<<63 {
			t.Fatal("broadcast lost signed zero")
		}
	}
}

func TestSIMDMemoryBounds(t *testing.T) {
	var v archsimd.Float32x4
	for _, tc := range []struct {
		name string
		f    func()
	}{
		{"short load", func() { archsimd.LoadFloat32x4(make([]float32, 3)) }},
		{"short store", func() { v.Store(make([]float32, 3)) }},
		{"nil array load", func() { memoryResult = archsimd.LoadFloat32x4Array(nil) }},
		{"nil array store", func() { v.StoreArray(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing memory bounds panic")
				}
			}()
			tc.f()
		})
	}
}
