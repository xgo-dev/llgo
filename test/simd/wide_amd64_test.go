//go:build goexperiment.simd && amd64

package simd_test

import (
	"math"
	"runtime"
	"simd/archsimd"
	"testing"
	"unsafe"
)

func requireSIMD256(t *testing.T) {
	t.Helper()
	// LLGo legalizes the baseline implementation without AVX. Official Go's
	// fixed-width API needs AVX2, so its control must run on eligible hardware.
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX2() {
		t.Skip("official Go SIMD256 requires AVX2")
	}
}

// Distinct upper/lower halves catch accidental 128-bit truncation. Exercise
// every power-of-two boundary at every lane width, as well as a random stream.
func testSIMD256Integer[T resizeInteger](t *testing.T, apply func([]T, []T, [][]T) (uint32, uint32)) {
	t.Helper()
	var elem T
	n := 32 / int(unsafe.Sizeof(elem))
	a, b := make([]T, n), make([]T, n)
	out := make([][]T, 8)
	for i := range out {
		out[i] = make([]T, n)
	}
	state := uint64(0xf123456789abcdef)
	for sample := 0; sample < 128; sample++ {
		for i := range a {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			a[i] = T(state)
			b[i] = T((uint64(1) << uint((sample+i)%64)) + uint64(sample%3) - 1)
			if i%5 == 0 {
				b[i] = a[i]
			}
		}
		equal, less := apply(a, b, out)
		var wantEqual, wantLess uint32
		for i := range a {
			x, y := a[i], b[i]
			wants := []T{x + y, x - y, x & y, x | y, x ^ y, x &^ y, ^x, -x}
			for op, want := range wants {
				if out[op][i] != want {
					t.Fatalf("sample %d operation %d lane %d: got %v want %v", sample, op, i, out[op][i], want)
				}
			}
			if x == y {
				wantEqual |= 1 << uint(i)
			}
			if x < y {
				wantLess |= 1 << uint(i)
			}
		}
		if equal != wantEqual || less != wantLess {
			t.Fatalf("mask got %#x/%#x want %#x/%#x", equal, less, wantEqual, wantLess)
		}
	}
}

func TestSIMD256Integers(t *testing.T) {
	requireSIMD256(t)
	t.Run("Int8x32", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []int8, out [][]int8) (uint32, uint32) {
			x, y := archsimd.LoadInt8x32(a), archsimd.LoadInt8x32(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			x.Neg().Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
	t.Run("Uint8x32", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []uint8, out [][]uint8) (uint32, uint32) {
			x, y := archsimd.LoadUint8x32(a), archsimd.LoadUint8x32(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			archsimd.BroadcastUint8x32(0).Sub(x).Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
	t.Run("Int16x16", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []int16, out [][]int16) (uint32, uint32) {
			x, y := archsimd.LoadInt16x16(a), archsimd.LoadInt16x16(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			x.Neg().Store(out[7])
			return simd256Mask16Bits(x.Equal(y)), simd256Mask16Bits(x.Less(y))
		})
	})
	t.Run("Uint16x16", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []uint16, out [][]uint16) (uint32, uint32) {
			x, y := archsimd.LoadUint16x16(a), archsimd.LoadUint16x16(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			archsimd.BroadcastUint16x16(0).Sub(x).Store(out[7])
			return simd256Mask16Bits(x.Equal(y)), simd256Mask16Bits(x.Less(y))
		})
	})
	t.Run("Int32x8", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []int32, out [][]int32) (uint32, uint32) {
			x, y := archsimd.LoadInt32x8(a), archsimd.LoadInt32x8(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			x.Neg().Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
	t.Run("Uint32x8", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []uint32, out [][]uint32) (uint32, uint32) {
			x, y := archsimd.LoadUint32x8(a), archsimd.LoadUint32x8(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			archsimd.BroadcastUint32x8(0).Sub(x).Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
	t.Run("Int64x4", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []int64, out [][]int64) (uint32, uint32) {
			x, y := archsimd.LoadInt64x4(a), archsimd.LoadInt64x4(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			x.Neg().Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
	t.Run("Uint64x4", func(t *testing.T) {
		testSIMD256Integer(t, func(a, b []uint64, out [][]uint64) (uint32, uint32) {
			x, y := archsimd.LoadUint64x4(a), archsimd.LoadUint64x4(b)
			x.Add(y).Store(out[0])
			x.Sub(y).Store(out[1])
			x.And(y).Store(out[2])
			x.Or(y).Store(out[3])
			x.Xor(y).Store(out[4])
			x.AndNot(y).Store(out[5])
			x.Not().Store(out[6])
			archsimd.BroadcastUint64x4(0).Sub(x).Store(out[7])
			return uint32(x.Equal(y).ToBits()), uint32(x.Less(y).ToBits())
		})
	})
}

func TestSIMD256Float32(t *testing.T) {
	requireSIMD256(t)
	var a, b, out [8]float32
	for i := range a {
		a[i] = float32(i) - 3.5
		b[i] = float32(i) + 0.5
	}
	x, y := archsimd.LoadFloat32x8Array(&a), archsimd.LoadFloat32x8Array(&b)
	for op, v := range []archsimd.Float32x8{x.Add(y), x.Sub(y), x.Mul(y), x.Div(y), x.Neg()} {
		v.StoreArray(&out)
		for i, got := range out {
			want := []float32{a[i] + b[i], a[i] - b[i], a[i] * b[i], a[i] / b[i], -a[i]}[op]
			if got != want {
				t.Fatalf("operation %d lane %d got %g want %g", op, i, got, want)
			}
		}
	}
	a[0], a[1], a[2], a[3] = float32(math.NaN()), float32(math.Inf(1)), float32(math.Copysign(0, -1)), 0
	x = archsimd.LoadFloat32x8Array(&a)
	var eq, ne uint8
	for i := range a {
		if a[i] == a[i] {
			eq |= 1 << uint(i)
		}
		if a[i] != a[i] {
			ne |= 1 << uint(i)
		}
	}
	if x.Equal(x).ToBits() != eq || x.NotEqual(x).ToBits() != ne {
		t.Fatal("unordered float comparison")
	}
	x.ToBits().BitsToFloat32().StoreArray(&out)
	for i := range a {
		if math.Float32bits(a[i]) != math.Float32bits(out[i]) {
			t.Fatal("bitcast changed floating bits")
		}
	}
}

func TestSIMD256Float64(t *testing.T) {
	requireSIMD256(t)
	var a, b, out [4]float64
	for i := range a {
		a[i] = float64(i) - 3.5
		b[i] = float64(i) + 0.5
	}
	x, y := archsimd.LoadFloat64x4Array(&a), archsimd.LoadFloat64x4Array(&b)
	for op, v := range []archsimd.Float64x4{x.Add(y), x.Sub(y), x.Mul(y), x.Div(y), x.Neg()} {
		v.StoreArray(&out)
		for i, got := range out {
			want := []float64{a[i] + b[i], a[i] - b[i], a[i] * b[i], a[i] / b[i], -a[i]}[op]
			if got != want {
				t.Fatalf("operation %d lane %d got %g want %g", op, i, got, want)
			}
		}
	}
	a[0], a[1], a[2], a[3] = float64(math.NaN()), float64(math.Inf(1)), float64(math.Copysign(0, -1)), 0
	x = archsimd.LoadFloat64x4Array(&a)
	var eq, ne uint8
	for i := range a {
		if a[i] == a[i] {
			eq |= 1 << uint(i)
		}
		if a[i] != a[i] {
			ne |= 1 << uint(i)
		}
	}
	if x.Equal(x).ToBits() != eq || x.NotEqual(x).ToBits() != ne {
		t.Fatal("unordered float comparison")
	}
	x.ToBits().BitsToFloat64().StoreArray(&out)
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(out[i]) {
			t.Fatal("bitcast changed floating bits")
		}
	}
}

func TestSIMD256MaskBits(t *testing.T) {
	requireSIMD256(t)
	if runtime.Compiler != "llgo" && !archsimd.X86.AVX512() {
		t.Skip("official mask bitmap constructors require AVX512")
	}
	for _, bits := range []uint32{0, 1, 0x80000000, 0x80018001, 0xa5a55a5a, 0xffffffff} {
		a := archsimd.Mask8x32FromBits(bits)
		if a.ToBits() != bits {
			t.Fatalf("32 lanes: %#x", a.ToBits())
		}
		b := archsimd.Mask8x32FromBits(^bits)
		if a.And(b).ToBits() != 0 || a.Or(b).ToBits() != ^uint32(0) {
			t.Fatal("32-lane mask logic")
		}
		if archsimd.Mask16x16FromBits(uint16(bits)).ToBits() != uint16(bits) {
			t.Fatal("16 lanes")
		}
		if archsimd.Mask32x8FromBits(uint8(bits)).ToBits() != uint8(bits) {
			t.Fatal("8 lanes")
		}
		if archsimd.Mask64x4FromBits(uint8(bits)).ToBits() != uint8(bits)&15 {
			t.Fatal("4 lanes")
		}
	}
}

func TestSIMD256MemoryAndLanes(t *testing.T) {
	requireSIMD256(t)
	backing := [10]float32{101, 1, 2, 3, 4, 5, 6, 7, 8, 202}
	x := archsimd.LoadFloat32x8(backing[1:])
	for i := uint8(0); i < 8; i++ {
		if func() float32 { var lanes [8]float32; x.StoreArray(&lanes); return lanes[i] }() != float32(i+1) {
			t.Fatal("load/lane")
		}
	}
	x = x.SetLo(x.GetLo().SetElem(0, -3)).SetHi(x.GetHi().SetElem(3, 77))
	x.Store(backing[1:])
	if backing[0] != 101 || backing[9] != 202 || backing[1] != -3 || backing[8] != 77 {
		t.Fatalf("store/sentinels %v", backing)
	}
	for _, f := range []func(){func() { archsimd.LoadFloat32x8(backing[:7]) }, func() { x.Store(backing[:7]) }} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("missing bounds panic")
				}
			}()
			f()
		}()
	}
	for mask := uint16(0); mask < 256; mask++ {
		out := [8]float32{9, 9, 9, 9, 9, 9, 9, 9}
		var lanes [8]int32
		for i := range lanes {
			if mask&(1<<uint(i)) != 0 {
				lanes[i] = 1
			}
		}
		x.StoreArrayMasked(&out, archsimd.LoadInt32x8Array(&lanes).Greater(archsimd.BroadcastInt32x8(0)))
		for i, got := range out {
			want := float32(9)
			if mask&(1<<uint(i)) != 0 {
				want = backing[i+1]
			}
			if got != want {
				t.Fatalf("mask %x lane %d got %g want %g", mask, i, got, want)
			}
		}
	}
}

func simd256Mask16Bits(mask archsimd.Mask16x16) uint32 {
	if runtime.Compiler == "llgo" || archsimd.X86.AVX512() {
		return uint32(mask.ToBits())
	}
	// The official ToBits instruction needs AVX512 even for this AVX2 mask.
	var lanes [16]int16
	mask.ToInt16x16().StoreArray(&lanes)
	var result uint32
	for i, v := range lanes {
		if v != 0 {
			result |= 1 << uint(i)
		}
	}
	return result
}
