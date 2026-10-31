//go:build cgo && (amd64 || arm64)

package cinterop

import (
	"math"
	"testing"
	"unsafe"
)

var importedVectorGlobal = cFloat4{13, 17, 19, 23}
var importedByteGlobal = cBytes16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

type ordinaryArray cFloat4

func TestSIMDCInteropImportedValues(t *testing.T) {
	a := makeFloat4(1, 2, 3, 4)
	b := makeFloat4(10, 20, 30, 40)
	if got := addFloat4(a, b); got != (cFloat4{11, 22, 33, 44}) {
		t.Fatalf("add = %v", got)
	}
	if got := manyVectors(a, b); got != (cFloat4{325, 650, 975, 1300}) {
		t.Fatalf("ten arguments = %v", got)
	}
	for i, want := range []float32{1, 2, 3, 4} {
		if got := laneFloat4(a, i); got != want {
			t.Fatalf("lane %d = %v", i, got)
		}
	}
	if got := xorUint4(cUint4{1, 0x80000000, 0xffffffff, 0x12345678}, cUint4{2, 0x80000001, 0x12345678, 0xffffffff}); got != (cUint4{3, 1, 0xedcba987, 0xedcba987}) {
		t.Fatalf("xor = %v", got)
	}
	if got := subDouble2(cDouble2{3.5, -9.25}, cDouble2{2, 8}); got != (cDouble2{1.5, -17.25}) {
		t.Fatalf("double = %v", got)
	}
	if got := addBytes16(importedByteGlobal); got != (cBytes16{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 21, 23, 25, 27, 29, 31}) {
		t.Fatalf("bytes = %v", got)
	}
	if got := aliasFloat4(a); got != a {
		t.Fatalf("typedef alias = %v", got)
	}
	if globalFloat4() != (cFloat4{3, 5, 7, 11}) || importedVectorGlobal != (cFloat4{13, 17, 19, 23}) {
		t.Fatal("global initialization")
	}
	var zero cFloat4
	if zero != (cFloat4{}) {
		t.Fatal("zero vector")
	}
	a[2] = 37
	if laneFloat4(a, 2) != 37 {
		t.Fatal("indexed store")
	}
	plain := ordinaryArray(a)
	if cFloat4(plain) != a || [4]cFloat(a) != [4]cFloat(plain) {
		t.Fatal("array conversion")
	}
	ptr := &a[1]
	*ptr = 53
	if laneFloat4(a, 1) != 53 {
		t.Fatal("lane address")
	}
	copy(a[:], []cFloat{7, 11, 13, 17})
	if a != (cFloat4{7, 11, 13, 17}) {
		t.Fatal("vector slicing")
	}
	withNaN := cFloat4{cFloat(math.NaN()), 1, 2, 3}
	if withNaN == withNaN {
		t.Fatal("array equality lost NaN semantics")
	}
	for _, index := range []int{-1, 4} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("index %d did not panic", index)
				}
			}()
			_ = a[index]
		}()
	}
}

func TestSIMDCInteropUnalignedSlices(t *testing.T) {
	var bytes [64]cByte
	start := int((1 - uintptr(unsafe.Pointer(&bytes[0]))) & 15)
	src := bytes[start : start+16]
	if uintptr(unsafe.Pointer(&src[0]))%16 != 1 {
		t.Fatal("byte test source is not deliberately unaligned")
	}
	want := cBytes16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	copy(src, want[:])
	if got := bytesFromSlice(src); got != want {
		t.Fatalf("slice copy = %v", got)
	}
	p := byteVectorPointer(src)
	if got := loadByteVector(p); got != want {
		t.Fatalf("pointer load = %v", got)
	}
	want[7] = 99
	storeByteVector(p, want)
	if src[7] != 99 || bytesFromSlice(src) != want {
		t.Fatal("pointer store did not update original slice")
	}
	var floats [16]cFloat
	floatStart := int(((4 - uintptr(unsafe.Pointer(&floats[0]))) & 15) / 4)
	fs := floats[floatStart : floatStart+4]
	if uintptr(unsafe.Pointer(&fs[0]))%16 != 4 {
		t.Fatal("float test source is not deliberately unaligned")
	}
	fwant := cFloat4{1, 2, 3, 4}
	copy(fs, fwant[:])
	fp := floatVectorPointer(fs)
	if floatsFromSlice(fs) != fwant || loadFloatVector(fp) != fwant {
		t.Fatal("unaligned float load")
	}
	fwant[2] = 37
	storeFloatVector(fp, fwant)
	if fs[2] != 37 || floatsFromSlice(fs) != fwant {
		t.Fatal("unaligned float store")
	}
	for _, convert := range []func(){
		func() { _ = bytesFromSlice(src[:15]) },
		func() { _ = byteVectorPointer(src[:15]) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("short slice conversion did not panic")
				}
			}()
			convert()
		}()
	}
}

func TestSIMDCInteropImportedAggregates(t *testing.T) {
	a := makeFloat4(1, 2, 3, 4)
	b := makeFloat4(10, 20, 30, 40)
	pair := cPair{a: a, b: b}
	if got := sumPair(pair); got != 3030 {
		t.Fatalf("pair = %v, want 3030", got)
	}
	if got := swapPair(pair); got.a != b || got.b != a {
		t.Fatalf("pair return = %v", got)
	}
	mixed := cMixed{a: a, b: cBytes16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}}
	changed := changeMixed(mixed)
	if changed.a != (cFloat4{2, 4, 6, 8}) || changed.b != (cBytes16{16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16}) {
		t.Fatalf("mixed vector shapes = %v", changed)
	}
	triple := cTriple{v: [3]cFloat4{a, b, importedVectorGlobal}}
	if got := rotateTriple(triple); got.v != [3]cFloat4{b, importedVectorGlobal, a} {
		t.Fatalf("nested vector array = %v", got)
	}
	box := cBox{tag: 5, v: a, tail: 6}
	if got := readBox(&box); got != 41 {
		t.Fatalf("box pointer = %v", got)
	}
	if got := changeBox(box); got.tag != 6 || got.tail != 8 || got.v != (cFloat4{2, 4, 6, 8}) {
		t.Fatalf("box value = %v", got)
	}
	if got := sumPlain(cPlainBox{v: [4]cFloat{1, 2, 3, 4}}); got != 30 {
		t.Fatalf("ordinary C array = %v", got)
	}
	if got := spillPair(a, b); got != 31020 {
		t.Fatalf("HVA spill = %v, want 31020", got)
	}
}
