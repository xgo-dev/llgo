//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func TestSIMDShiftAll(t *testing.T) {
	counts := []uint64{0, 1, 7, 8, 15, 16, 31, 32, 33, 63, 64, 65, 127, 128, 255, 256, 1 << 32, math.MaxUint64}
	x16 := [8]uint16{0, 1, 32767, 32768, 65535, 123, 9, 40000}
	x32 := [4]int32{math.MinInt32, math.MaxInt32, -1, 1}
	x64 := [2]uint64{math.MaxUint64, 1 << 63}
	for _, n := range counts {
		var left16, right16 [8]uint16
		v16 := archsimd.LoadUint16x8Array(&x16)
		v16.ShiftAllLeft(n).StoreArray(&left16)
		v16.ShiftAllRight(n).StoreArray(&right16)
		for i, x := range x16 {
			if left16[i] != x<<n || right16[i] != x>>n {
				t.Fatalf("uint16 shift %d lane %d", n, i)
			}
		}
		var left32, right32 [4]int32
		v32 := archsimd.LoadInt32x4Array(&x32)
		v32.ShiftAllLeft(n).StoreArray(&left32)
		v32.ShiftAllRight(n).StoreArray(&right32)
		for i, x := range x32 {
			if left32[i] != x<<n || right32[i] != x>>n {
				t.Fatalf("int32 shift %d lane %d", n, i)
			}
		}
		var left64, right64 [2]uint64
		v64 := archsimd.LoadUint64x2Array(&x64)
		v64.ShiftAllLeft(n).StoreArray(&left64)
		v64.ShiftAllRight(n).StoreArray(&right64)
		for i, x := range x64 {
			if left64[i] != x<<n || right64[i] != x>>n {
				t.Fatalf("uint64 shift %d lane %d", n, i)
			}
		}
	}
}

func TestSIMDSaturatedAndMinMax(t *testing.T) {
	x := [16]int8{-128, -127, -100, -1, 0, 1, 100, 127, -128, 127, -1, 1, 64, -64, 0, 42}
	y := [16]int8{-1, -127, -100, -128, 127, 127, 100, 1, 127, -128, 1, -1, 64, -64, 0, 42}
	a, b := archsimd.LoadInt8x16Array(&x), archsimd.LoadInt8x16Array(&y)
	var add, sub, lo, hi [16]int8
	a.AddSaturated(b).StoreArray(&add)
	a.SubSaturated(b).StoreArray(&sub)
	a.Min(b).StoreArray(&lo)
	a.Max(b).StoreArray(&hi)
	clamp := func(v int) int8 {
		if v < -128 {
			return -128
		}
		if v > 127 {
			return 127
		}
		return int8(v)
	}
	for i := range x {
		if add[i] != clamp(int(x[i])+int(y[i])) || sub[i] != clamp(int(x[i])-int(y[i])) || lo[i] != min(x[i], y[i]) || hi[i] != max(x[i], y[i]) {
			t.Fatalf("signed saturated/minmax lane %d: %d %d %d %d", i, add[i], sub[i], lo[i], hi[i])
		}
	}
	ux := [8]uint16{0, 1, 65535, 65534, 32768, 123, 9, 40000}
	uy := [8]uint16{1, 2, 1, 65535, 32768, 100, 10, 30000}
	ua, ub := archsimd.LoadUint16x8Array(&ux), archsimd.LoadUint16x8Array(&uy)
	var uadd, usub, ulo, uhi [8]uint16
	ua.AddSaturated(ub).StoreArray(&uadd)
	ua.SubSaturated(ub).StoreArray(&usub)
	ua.Min(ub).StoreArray(&ulo)
	ua.Max(ub).StoreArray(&uhi)
	for i := range ux {
		wantAdd := uint16(min(uint32(ux[i])+uint32(uy[i]), 65535))
		wantSub := uint16(max(int(ux[i])-int(uy[i]), 0))
		if uadd[i] != wantAdd || usub[i] != wantSub || ulo[i] != min(ux[i], uy[i]) || uhi[i] != max(ux[i], uy[i]) {
			t.Fatalf("unsigned saturated/minmax lane %d", i)
		}
	}
}
