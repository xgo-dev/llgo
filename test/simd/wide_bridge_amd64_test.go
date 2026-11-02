//go:build goexperiment.simd && amd64

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

func testSIMD256BridgeInteger[T resizeInteger](t *testing.T, apply func([]T, []T, [][]T)) {
	t.Helper()
	n := 32 / int(unsafe.Sizeof(T(0)))
	a, b := make([]T, n), make([]T, n)
	out := [][]T{make([]T, n), make([]T, n), make([]T, n)}
	state := uint64(0x84222325cbf29ce4)
	for sample := 0; sample < 128; sample++ {
		for i := range a {
			state ^= state << 13
			state ^= state >> 7
			state ^= state << 17
			a[i] = T(state)
			b[i] = T((uint64(1) << uint((sample+i)%64)) + uint64(sample%3) - 1)
		}
		apply(a, b, out)
		for i, x := range a {
			if out[0][i] != min(x, b[i]) || out[1][i] != max(x, b[i]) || out[2][i] != x {
				t.Fatalf("sample %d lane %d: min/max/roundtrip = %v/%v/%v for %v,%v", sample, i, out[0][i], out[1][i], out[2][i], x, b[i])
			}
		}
	}
}

func TestSIMD256BridgeInteger(t *testing.T) {
	requireSIMD256(t)
	t.Run("int8", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []int8, out [][]int8) {
			x, y := archsimd.LoadInt8x32(a), archsimd.LoadInt8x32(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToUint8().ConvertToInt8().Store(out[2])
		})
	})
	t.Run("uint8", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []uint8, out [][]uint8) {
			x, y := archsimd.LoadUint8x32(a), archsimd.LoadUint8x32(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToInt8().ConvertToUint8().Store(out[2])
		})
	})
	t.Run("int16", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []int16, out [][]int16) {
			x, y := archsimd.LoadInt16x16(a), archsimd.LoadInt16x16(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToUint16().ConvertToInt16().Store(out[2])
		})
	})
	t.Run("uint16", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []uint16, out [][]uint16) {
			x, y := archsimd.LoadUint16x16(a), archsimd.LoadUint16x16(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToInt16().ConvertToUint16().Store(out[2])
		})
	})
	t.Run("int32", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []int32, out [][]int32) {
			x, y := archsimd.LoadInt32x8(a), archsimd.LoadInt32x8(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToUint32().ConvertToInt32().Store(out[2])
		})
	})
	t.Run("uint32", func(t *testing.T) {
		testSIMD256BridgeInteger(t, func(a, b []uint32, out [][]uint32) {
			x, y := archsimd.LoadUint32x8(a), archsimd.LoadUint32x8(b)
			x.Min(y).Store(out[0])
			x.Max(y).Store(out[1])
			x.ConvertToInt32().ConvertToUint32().Store(out[2])
		})
	})
}

func TestSIMD256BridgeSaturation(t *testing.T) {
	requireSIMD256(t)
	// Every byte pair, including different values in both 128-bit groups.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b += 32 {
			var values [32]uint8
			for i := range values {
				values[i] = uint8(b + i)
			}
			x, y := archsimd.BroadcastUint8x32(uint8(a)), archsimd.LoadUint8x32Array(&values)
			var add, sub, average [32]uint8
			x.AddSaturated(y).StoreArray(&add)
			x.SubSaturated(y).StoreArray(&sub)
			x.Average(y).StoreArray(&average)
			var signedAdd, signedSub [32]int8
			x.ConvertToInt8().AddSaturated(y.ConvertToInt8()).StoreArray(&signedAdd)
			x.ConvertToInt8().SubSaturated(y.ConvertToInt8()).StoreArray(&signedSub)
			for i, v := range values {
				if add[i] != uint8(min(a+int(v), 255)) || sub[i] != uint8(max(a-int(v), 0)) || average[i] != uint8((a+int(v)+1)/2) {
					t.Fatalf("unsigned byte saturation/average: %d,%d lane %d", a, v, i)
				}
				sx, sy := int(int8(a)), int(int8(v))
				if signedAdd[i] != int8(min(max(sx+sy, -128), 127)) || signedSub[i] != int8(min(max(sx-sy, -128), 127)) {
					t.Fatalf("signed byte saturation: %d,%d lane %d", sx, sy, i)
				}
			}
		}
	}
	boundaries := []uint16{0, 1, 2, 32766, 32767, 32768, 32769, 65534, 65535}
	for _, a := range boundaries {
		for sample := range boundaries {
			var values [16]uint16
			for i := range values {
				values[i] = boundaries[(sample+i)%len(boundaries)]
			}
			x, y := archsimd.BroadcastUint16x16(a), archsimd.LoadUint16x16Array(&values)
			var add, sub, average [16]uint16
			x.AddSaturated(y).StoreArray(&add)
			x.SubSaturated(y).StoreArray(&sub)
			x.Average(y).StoreArray(&average)
			var signedAdd, signedSub [16]int16
			x.ConvertToInt16().AddSaturated(y.ConvertToInt16()).StoreArray(&signedAdd)
			x.ConvertToInt16().SubSaturated(y.ConvertToInt16()).StoreArray(&signedSub)
			for i, v := range values {
				if add[i] != uint16(min(int(a)+int(v), 65535)) || sub[i] != uint16(max(int(a)-int(v), 0)) || average[i] != uint16((uint32(a)+uint32(v)+1)/2) {
					t.Fatalf("unsigned word saturation/average: %d,%d lane %d", a, v, i)
				}
				sx, sy := int(int16(a)), int(int16(v))
				if signedAdd[i] != int16(min(max(sx+sy, -32768), 32767)) || signedSub[i] != int16(min(max(sx-sy, -32768), 32767)) {
					t.Fatalf("signed word saturation: %d,%d lane %d", sx, sy, i)
				}
			}
		}
	}
}

func TestSIMD256BridgeSqrt(t *testing.T) {
	requireSIMD256(t)
	values := []float64{0, math.Copysign(0, -1), 2, 4, -1, math.SmallestNonzeroFloat64, math.Inf(1), math.NaN(), math.MaxFloat64}
	for start := range values {
		var a [4]float64
		var b [8]float32
		for i := range a {
			a[i] = values[(start+i)%len(values)]
		}
		for i := range b {
			b[i] = float32(values[(start+i)%len(values)])
		}
		var out64 [4]float64
		var out32 [8]float32
		archsimd.LoadFloat64x4Array(&a).Sqrt().StoreArray(&out64)
		archsimd.LoadFloat32x8Array(&b).Sqrt().StoreArray(&out32)
		for i, x := range a {
			want := math.Sqrt(x)
			if math.Float64bits(out64[i]) != math.Float64bits(want) && !(math.IsNaN(out64[i]) && math.IsNaN(want)) {
				t.Fatalf("sqrt64 lane %d = %g, want %g", i, out64[i], want)
			}
		}
		for i, x := range b {
			want := float32(math.Sqrt(float64(x)))
			if math.Float32bits(out32[i]) != math.Float32bits(want) && !(math.IsNaN(float64(out32[i])) && math.IsNaN(float64(want))) {
				t.Fatalf("sqrt32 lane %d = %g, want %g", i, out32[i], want)
			}
		}
	}
}

func TestSIMD256BridgeCalls(t *testing.T) {
	requireSIMD256(t)
	a := [8]float32{1, 4, 9, 16, 25, 36, 49, 64}
	x := archsimd.LoadFloat32x8Array(&a)
	method := x.Sqrt
	x = archsimd.BroadcastFloat32x8(100)
	checkSIMD256Lanes(t, method(), func(i int) float32 { return float32(i + 1) })
	x = archsimd.LoadFloat32x8Array(&a)
	checkSIMD256Lanes(t, simd256Indirect(archsimd.Float32x8.Min, x, archsimd.BroadcastFloat32x8(30)), func(i int) float32 { return min(a[i], 30) })
	defer func() {
		checkSIMD256Lanes(t, method(), func(i int) float32 { return float32(i + 1) })
	}()
	result, integers := vectorcall.WideBridge(x)
	checkSIMD256Lanes(t, result, func(i int) float32 { return float32(i + 1) })
	var got [8]int32
	integers.StoreArray(&got)
	for i, v := range got {
		if v != int32(i+1) {
			t.Fatalf("cross-package converted lane %d = %d", i, v)
		}
	}
}
