//go:build llgo && goexperiment.simd && amd64

package portable

import (
	"math"
	"simd"
	"testing"
)

// Qualify the amd64 portable256 bridge at every selectable width.
// Distinct groups catch accidentally selecting a 128-bit bridge for a wider
// portable representation. Scalar references also work with CPU features off.
func TestPortableBridgeCarryless(t *testing.T) {
	n := (simd.Uint64s{}).Len()
	a, b, out := make([]uint64, n), make([]uint64, n), make([]uint64, n)
	for i := range a {
		a[i] = uint64(i+1) * 0x123456789abcdef
		b[i] = uint64(i+3) * 0xfedcba987654321
	}
	x, y := simd.LoadUint64s(a), simd.LoadUint64s(b)
	for part := 0; part < 2; part++ {
		if part == 0 {
			x.CarrylessMultiplyEven(y).Store(out)
		} else {
			x.CarrylessMultiplyOdd(y).Store(out)
		}
		for base := 0; base < n; base += 2 {
			var lo, hi uint64
			for bit := 0; bit < 64; bit++ {
				if (a[base+part]>>bit)&1 != 0 {
					lo ^= b[base+part] << bit
					if bit != 0 {
						hi ^= b[base+part] >> (64 - bit)
					}
				}
			}
			if out[base] != lo || out[base+1] != hi {
				t.Fatalf("part=%d group=%d got=%x want=[%x %x]", part, base/2, out[base:base+2], lo, hi)
			}
		}
	}
}

func TestPortableBridgeSaturationAndFMA(t *testing.T) {
	n := (simd.Uint8s{}).Len()
	a, b, out := make([]uint8, n), make([]uint8, n), make([]uint8, n)
	for i := range a {
		a[i] = uint8(230 + i)
		b[i] = uint8(3 * i)
	}
	simd.LoadUint8s(a).AddSaturated(simd.LoadUint8s(b)).Store(out)
	for i, v := range out {
		want := min(int(a[i])+int(b[i]), 255)
		if int(v) != want {
			t.Fatalf("saturation lane %d=%d want %d", i, v, want)
		}
	}
	// The exact product differs from one by 2^-46: rounding before adding
	// would lose the residual. Hardware-width bridges must retain the fused
	// result even with FMA disabled. Go's emulated bridge is ordinary x*y+z,
	// which may round the intermediate product or be fused by the compiler.
	f := make([]float32, (simd.Float32s{}).Len())
	simd.BroadcastFloat32s(1+0x1p-23).MulAdd(simd.BroadcastFloat32s(1-0x1p-23), simd.BroadcastFloat32s(-1)).Store(f)
	for i, v := range f {
		fused := math.Float32bits(v) == math.Float32bits(-0x1p-46)
		separate := simd.Emulated() && math.Float32bits(v) == 0
		if !fused && !separate {
			t.Fatalf("FMA lane %d=%g want %g", i, v, float32(-0x1p-46))
		}
	}
}
