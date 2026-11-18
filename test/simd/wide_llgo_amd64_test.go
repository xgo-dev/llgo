//go:build goexperiment.simd && llgo && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
	_ "unsafe"
)

//go:linkname simd256LinkedAdd simd/archsimd.Float32x8.Add
func simd256LinkedAdd(x, y archsimd.Float32x8) archsimd.Float32x8

func TestSIMD256IntrinsicBoundary(t *testing.T) {
	x, y := archsimd.BroadcastFloat32x8(7), archsimd.BroadcastFloat32x8(2)
	checkSIMD256Lanes(t, simd256LinkedAdd(x, y), func(int) float32 { return 9 })
	for _, call := range []func(){
		func() { var p *[8]float32; archsimd.LoadFloat32x8Array(p) },
		func() { x.StoreArray(nil) },
		func() { x.StoreArrayMasked(nil, archsimd.Mask32x8FromBits(128)) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing active nil panic")
				}
			}()
			call()
		}()
	}
	// The official Uint64x4.Mul needs AVX512. LLGo's generic vector multiply
	// must retain its wraparound semantics even with every optional flag off.
	a := [4]uint64{0, 1 << 63, ^uint64(0), 0xfedcba9876543210}
	b := [4]uint64{99, 3, 17, 0x0123456789abcdef}
	var out [4]uint64
	archsimd.LoadUint64x4Array(&a).Mul(archsimd.LoadUint64x4Array(&b)).StoreArray(&out)
	for i, v := range out {
		if v != a[i]*b[i] {
			t.Fatalf("wide multiply lane %d got %x want %x", i, v, a[i]*b[i])
		}
	}
}

//go:noinline
func simd256Defer(x, y archsimd.Float32x8) (result archsimd.Float32x8) {
	defer func() {
		if recover() == nil {
			panic("missing recovery")
		}
		result = x.Sub(y)
	}()
	panic("wide value remains live")
}

func TestSIMD256Recovery(t *testing.T) {
	x, y := archsimd.BroadcastFloat32x8(9), archsimd.BroadcastFloat32x8(2)
	checkSIMD256Lanes(t, simd256Defer(x, y), func(int) float32 { return 7 })
}
