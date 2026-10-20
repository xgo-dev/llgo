//go:build llgo && goexperiment.simd && (amd64 || arm64)

package cinterop

import (
	"simd/archsimd"
	"testing"
	"unsafe"
)

var globalNativeBox = nativeBox{tag: 7, tail: 9}

func TestSIMDCInteropDirect(t *testing.T) {
	a := archsimd.LoadFloat32x4Array(&[4]float32{1, 2, 3, 4})
	b := archsimd.LoadFloat32x4Array(&[4]float32{10, 20, 30, 40})
	check := func(name string, v archsimd.Float32x4, want [4]float32) {
		t.Helper()
		var got [4]float32
		v.StoreArray(&got)
		if got != want {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
	}
	check("direct", directAdd(a, b), [4]float32{11, 22, 33, 44})
	check("C result in Go SIMD", directAdd(a, b).Mul(a), [4]float32{11, 44, 99, 176})
	check("callback", directCallback(vectorSubtract, a, b), [4]float32{-9, -18, -27, -36})
	check("export", directExport(a, b), [4]float32{-9, -18, -27, -36})
	pair := directPair(nativePair{a, b})
	check("HVA first", pair.a, [4]float32{10, 20, 30, 40})
	check("HVA second", pair.b, [4]float32{1, 2, 3, 4})
	pair = directPairCallback(swapNativePair, nativePair{a, b})
	check("HVA callback first", pair.a, [4]float32{10, 20, 30, 40})
	check("HVA callback second", pair.b, [4]float32{1, 2, 3, 4})
	var box nativeBox
	if unsafe.Sizeof(box) != 48 || unsafe.Alignof(box) != 16 || unsafe.Offsetof(box.v) != 16 {
		t.Fatal("native SIMD struct source layout")
	}
	box = nativeBox{7, a, 9}
	globalNativeBox.v = a
	heap := new(nativeBox)
	*heap = box
	nested := struct {
		prefix byte
		boxes  [2]nativeBox
	}{boxes: [2]nativeBox{box, box}}
	for _, p := range []*nativeBox{&box, &globalNativeBox, heap, &nested.boxes[1]} {
		if offset := uintptr(unsafe.Pointer(&p.v)) - uintptr(unsafe.Pointer(p)); offset != unsafe.Offsetof(box.v) {
			t.Fatalf("actual offset %d", offset)
		}
		if got := directReadBox(p); got != 46 {
			t.Fatalf("C pointer read = %v", got)
		}
		changed := directBox(*p)
		if changed.tag != 8 || changed.tail != 11 {
			t.Fatal("native struct scalar fields")
		}
		check("native struct return", changed.v, [4]float32{2, 4, 6, 8})
	}
	ordinary := struct {
		tag  byte
		v    archsimd.Float32x4
		tail byte
	}{}
	if unsafe.Sizeof(ordinary) != 48 || unsafe.Alignof(ordinary) != 16 || unsafe.Offsetof(ordinary.v) != 16 {
		t.Fatal("ordinary Go SIMD storage is not natively aligned")
	}
	ordinary.tag, ordinary.v, ordinary.tail = 7, a, 9
	if got := directReadBox((*nativeBox)(unsafe.Pointer(&ordinary))); got != 46 {
		t.Fatalf("C read of ordinary Go SIMD record = %v", got)
	}
	var embedded nativeEmbedded
	if unsafe.Sizeof(embedded) != 48 || unsafe.Alignof(embedded) != 16 || unsafe.Offsetof(embedded.v) != 16 {
		t.Fatal("native layout with an embedded pointer")
	}
	if uintptr(unsafe.Pointer(&embedded.v))-uintptr(unsafe.Pointer(&embedded)) != unsafe.Offsetof(embedded.v) {
		t.Fatal("embedded pointer source offset differs from storage")
	}
}
