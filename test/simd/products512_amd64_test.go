//go:build goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMD512Products(t *testing.T) {
	if !archsimd.X86.AVX512() {
		t.Skip("requires AVX512")
	}
	var a [64]uint8
	var b [64]int8
	for i := range a {
		a[i] = 255
		b[i] = 127
		if i%4 >= 2 {
			b[i] = -128
		}
	}
	x := archsimd.LoadUint8x64Array(&a)
	y := archsimd.LoadInt8x64Array(&b)
	var out [32]int16
	x.DotProductPairsSaturated(y).StoreArray(&out)
	for i, got := range out {
		want := int16(32767)
		if i%2 != 0 {
			want = -32768
		}
		if got != want {
			t.Fatalf("product pair %d = %d, want %d", i, got, want)
		}
	}
	var f [16]float32
	for i := range f {
		f[i] = float32(i + 2)
	}
	v := archsimd.LoadFloat32x16Array(&f)
	var p, q [16]float32
	v.MulAddEvenSubOdd(v, v).StoreArray(&p)
	v.MulAddOddSubEven(v, v).StoreArray(&q)
	for i, n := range f {
		add := n
		if i%2 == 0 {
			add = -add
		}
		if p[i] != n*n+add || q[i] != n*n-add {
			t.Fatalf("alternating FMA lane %d = %v/%v, want %v/%v", i, p[i], q[i], n*n+add, n*n-add)
		}
	}
}

func TestSIMDProductEdges(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("requires AVX2")
	}
	signs := [16]int8{-1, 0, 1, -1}
	values := [16]int8{-128, -128, -128, 127}
	var signed [16]int8
	archsimd.LoadInt8x16Array(&values).MulSign(archsimd.LoadInt8x16Array(&signs)).StoreArray(&signed)
	if want := ([16]int8{-128, 0, -128, -127}); signed != want {
		t.Fatalf("MulSign = %v, want %v", signed, want)
	}

	a := [8]int16{-32768, -32768, 32767, 32767, -1, 3, 4, -5}
	b := [8]int16{-32768, 32767, -32768, 32767, 8, -6, 7, 9}
	x, y := archsimd.LoadInt16x8Array(&a), archsimd.LoadInt16x8Array(&b)
	var hi [8]int16
	var pair [4]int32
	var uhi [8]uint16
	x.MulHigh(y).StoreArray(&hi)
	x.AsUint16x8().MulHigh(y.AsUint16x8()).StoreArray(&uhi)
	x.DotProductPairs(y).StoreArray(&pair)
	for i := range a {
		prod := int32(a[i]) * int32(b[i])
		uprod := uint32(uint16(a[i])) * uint32(uint16(b[i]))
		if hi[i] != int16(prod>>16) || uhi[i] != uint16(uprod>>16) {
			t.Fatalf("MulHigh lane %d = %d/%d, want %d/%d", i, hi[i], uhi[i], prod>>16, uprod>>16)
		}
		if i%2 == 0 && pair[i/2] != prod+int32(a[i+1])*int32(b[i+1]) {
			t.Fatalf("DotProductPairs lane %d = %d", i/2, pair[i/2])
		}
	}

	words := [8]int32{-1, 18, -2147483648, 19, 2147483647, 20, -123456789, 21}
	w := archsimd.LoadInt32x8Array(&words)
	var sw [4]int64
	var uw [4]uint64
	w.MulWidenEven(w).StoreArray(&sw)
	w.AsUint32x8().MulWidenEven(w.AsUint32x8()).StoreArray(&uw)
	for i := range sw {
		s, u := int64(words[2*i]), uint64(uint32(words[2*i]))
		if sw[i] != s*s || uw[i] != u*u {
			t.Fatalf("MulWidenEven lane %d = %d/%d", i, sw[i], uw[i])
		}
	}

	var left, right [32]uint8
	for i := range left {
		left[i], right[i] = uint8(9*i), uint8(255-5*i)
	}
	var sums [4]uint64
	archsimd.LoadUint8x32Array(&left).SumOf8AbsDiff(archsimd.LoadUint8x32Array(&right)).StoreArray(&sums)
	for i, got := range sums {
		var want uint64
		for j := 8 * i; j < 8*i+8; j++ {
			d := int(left[j]) - int(right[j])
			if d < 0 {
				d = -d
			}
			want += uint64(d)
		}
		if got != want {
			t.Fatalf("SumOf8AbsDiff group %d = %d, want %d", i, got, want)
		}
	}
	floats := [4]float32{1, 2, 3, 4}
	f := archsimd.LoadFloat32x4Array(&floats)
	var alt [4]float32
	f.AddOddSubEven(f).StoreArray(&alt)
	if alt != ([4]float32{0, 4, 0, 8}) {
		t.Fatal("AddOddSubEven", alt)
	}
}

func TestSIMDSaturatedHorizontalPairs(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("requires AVX2")
	}
	a := [16]int16{32767, 1, -32768, -1, 32767, -1, -32768, 1, 10, 11, 12, 13, 14, 15, 16, 17}
	b := [16]int16{7, 9, 20, -7, 3, -4, -10, 8, 32767, 3, -32768, -3, 32767, -2, -32768, 2}
	x, y := archsimd.LoadInt16x16Array(&a), archsimd.LoadInt16x16Array(&b)
	var add, sub [16]int16
	x.ConcatAddPairsSaturatedGrouped(y).StoreArray(&add)
	x.ConcatSubPairsSaturatedGrouped(y).StoreArray(&sub)
	var loA, loB, loAdd, loSub [8]int16
	copy(loA[:], a[:])
	copy(loB[:], b[:])
	u, v := archsimd.LoadInt16x8Array(&loA), archsimd.LoadInt16x8Array(&loB)
	u.ConcatAddPairsSaturated(v).StoreArray(&loAdd)
	u.ConcatSubPairsSaturated(v).StoreArray(&loSub)
	for i := range add {
		input := a
		if i%8 >= 4 {
			input = b
		}
		j := i/8*8 + (i%4)*2
		l, r := int(input[j]), int(input[j+1])
		wantAdd, wantSub := int16(max(-32768, min(32767, l+r))), int16(max(-32768, min(32767, l-r)))
		if add[i] != wantAdd || sub[i] != wantSub || (i < 8 && (loAdd[i] != wantAdd || loSub[i] != wantSub)) {
			t.Fatalf("lane %d: grouped %v/%v plain %v/%v, want %d/%d", i, add, sub, loAdd, loSub, wantAdd, wantSub)
		}
	}
}
