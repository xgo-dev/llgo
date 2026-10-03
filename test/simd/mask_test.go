//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"math"
	"simd/archsimd"
	"testing"
)

func checkMask32(t *testing.T, name string, mask archsimd.Mask32x4, want [4]bool) {
	t.Helper()
	var got [4]int32
	mask.ToInt32x4().StoreArray(&got)
	for i, yes := range want {
		value := int32(0)
		if yes {
			value = -1
		}
		if got[i] != value {
			t.Fatalf("%s lane %d = %x want %x", name, i, got[i], value)
		}
	}
}

func TestSIMDCompare(t *testing.T) {
	x, y := [4]int32{math.MinInt32, -1, 7, math.MaxInt32}, [4]int32{math.MaxInt32, -1, -8, math.MinInt32}
	a, b := archsimd.LoadInt32x4Array(&x), archsimd.LoadInt32x4Array(&y)
	checkMask32(t, "signed equal", a.Equal(b), [4]bool{false, true, false, false})
	checkMask32(t, "signed not equal", a.NotEqual(b), [4]bool{true, false, true, true})
	checkMask32(t, "signed less", a.Less(b), [4]bool{true, false, false, false})
	checkMask32(t, "signed less equal", a.LessEqual(b), [4]bool{true, true, false, false})
	checkMask32(t, "signed greater", a.Greater(b), [4]bool{false, false, true, true})
	checkMask32(t, "signed greater equal", a.GreaterEqual(b), [4]bool{false, true, true, true})
	checkMask32(t, "unsigned less", a.ToBits().Less(b.ToBits()), [4]bool{false, false, true, true})
	checkMask32(t, "unsigned greater equal", a.ToBits().GreaterEqual(b.ToBits()), [4]bool{true, true, false, false})
	fx, fy := [4]float32{float32(math.NaN()), float32(math.Inf(1)), math.Float32frombits(1 << 31), 3}, [4]float32{1, float32(math.Inf(1)), 0, -4}
	fa, fb := archsimd.LoadFloat32x4Array(&fx), archsimd.LoadFloat32x4Array(&fy)
	checkMask32(t, "float equal", fa.Equal(fb), [4]bool{false, true, true, false})
	checkMask32(t, "float not equal", fa.NotEqual(fb), [4]bool{true, false, false, true})
	checkMask32(t, "float less", fa.Less(fb), [4]bool{})
	checkMask32(t, "float less equal", fa.LessEqual(fb), [4]bool{false, true, true, false})
	checkMask32(t, "float greater", fa.Greater(fb), [4]bool{false, false, false, true})
	checkMask32(t, "float greater equal", fa.GreaterEqual(fb), [4]bool{false, true, true, true})
	checkMask32(t, "NaN rhs", fb.LessEqual(fa), [4]bool{false, true, true, true})
}

//go:noinline
func passMask(m archsimd.Mask32x4) (archsimd.Mask32x4, int) { return m, 37 }

func TestSIMDMaskStorageAndSelect(t *testing.T) {
	values := [4]int32{0, 1, -1, math.MinInt32}
	m := archsimd.LoadInt32x4Array(&values).ToMask()
	checkMask32(t, "nonzero", m, [4]bool{false, true, true, true})
	type record struct {
		before byte
		masks  [2]archsimd.Mask32x4
		after  byte
	}
	r := record{before: 19, masks: [2]archsimd.Mask32x4{m, {}}, after: 23}
	var boxed any = r.masks[0]
	f := passMask
	got, n := f(boxed.(archsimd.Mask32x4))
	if r.before != 19 || r.after != 23 || n != 37 {
		t.Fatal("mask storage damaged adjacent data")
	}
	checkMask32(t, "indirect storage", got.And(m).Or(r.masks[1]), [4]bool{false, true, true, true})
	x := [4]float32{5, math.Float32frombits(0x7fc12345), math.Float32frombits(1 << 31), -7}
	y := [4]float32{-2, 99, 99, 99}
	a, b := archsimd.LoadFloat32x4Array(&x), archsimd.LoadFloat32x4Array(&y)
	var selected, masked [4]float32
	a.IfElse(m, b).StoreArray(&selected)
	a.Masked(m).StoreArray(&masked)
	for i := range x {
		want, wantMasked := x[i], x[i]
		if i == 0 {
			want, wantMasked = y[i], 0
		}
		if math.Float32bits(selected[i]) != math.Float32bits(want) || math.Float32bits(masked[i]) != math.Float32bits(wantMasked) {
			t.Fatalf("selection lane %d: %x %x", i, math.Float32bits(selected[i]), math.Float32bits(masked[i]))
		}
	}
	var zero archsimd.Mask32x4
	checkMask32(t, "zero mask", zero, [4]bool{})
}

func TestSIMDMaskWidths(t *testing.T) {
	var bytes [16]int8
	for i := range bytes {
		bytes[i] = int8(i) - 8
	}
	a := archsimd.LoadInt8x16Array(&bytes)
	var selected [16]int8
	a.IfElse(a.Greater(archsimd.BroadcastInt8x16(0)), archsimd.BroadcastInt8x16(42)).StoreArray(&selected)
	for i, x := range bytes {
		want := x
		if x <= 0 {
			want = 42
		}
		if selected[i] != want {
			t.Fatalf("byte lane %d", i)
		}
	}
	words := [8]uint16{0, 1, 32767, 32768, 65535, 9, 123, 1000}
	u := archsimd.LoadUint16x8Array(&words)
	var mask16 [8]int16
	u.Greater(archsimd.BroadcastUint16x8(32767)).ToInt16x8().StoreArray(&mask16)
	for i, x := range words {
		want := int16(0)
		if x > 32767 {
			want = -1
		}
		if mask16[i] != want {
			t.Fatalf("uint16 lane %d", i)
		}
	}
	longs := [2]int64{math.MinInt64, math.MaxInt64}
	v := archsimd.LoadInt64x2Array(&longs)
	var mask64 [2]int64
	v.Less(archsimd.BroadcastInt64x2(0)).ToInt64x2().StoreArray(&mask64)
	if mask64 != [2]int64{-1, 0} {
		t.Fatal(mask64)
	}
}
