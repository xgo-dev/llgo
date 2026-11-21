//go:build goexperiment.simd && wasm

package simd_test

import (
	"simd/archsimd"
	"testing"
)

func TestSIMDMulWidenHiInt8x16(t *testing.T) {
	var x, y [16]int8
	for i := range x {
		x[i] = int8(i*29 - 101)
		y[i] = int8(77 - i*19)
	}
	a, b := archsimd.LoadInt8x16Array(&x), archsimd.LoadInt8x16Array(&y)
	var got [8]int16
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := int16(x[i+8]) * int16(y[i+8]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
func TestSIMDMulWidenHiInt16x8(t *testing.T) {
	var x, y [8]int16
	for i := range x {
		x[i] = int16(i*29 - 101)
		y[i] = int16(77 - i*19)
	}
	a, b := archsimd.LoadInt16x8Array(&x), archsimd.LoadInt16x8Array(&y)
	var got [4]int32
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := int32(x[i+4]) * int32(y[i+4]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
func TestSIMDMulWidenHiInt32x4(t *testing.T) {
	var x, y [4]int32
	for i := range x {
		x[i] = int32(i*29 - 101)
		y[i] = int32(77 - i*19)
	}
	a, b := archsimd.LoadInt32x4Array(&x), archsimd.LoadInt32x4Array(&y)
	var got [2]int64
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := int64(x[i+2]) * int64(y[i+2]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
func TestSIMDMulWidenHiUint8x16(t *testing.T) {
	var x, y [16]uint8
	for i := range x {
		x[i] = uint8(i*29 - 101)
		y[i] = uint8(77 - i*19)
	}
	a, b := archsimd.LoadUint8x16Array(&x), archsimd.LoadUint8x16Array(&y)
	var got [8]uint16
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := uint16(x[i+8]) * uint16(y[i+8]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
func TestSIMDMulWidenHiUint16x8(t *testing.T) {
	var x, y [8]uint16
	for i := range x {
		x[i] = uint16(i*29 - 101)
		y[i] = uint16(77 - i*19)
	}
	a, b := archsimd.LoadUint16x8Array(&x), archsimd.LoadUint16x8Array(&y)
	var got [4]uint32
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := uint32(x[i+4]) * uint32(y[i+4]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
func TestSIMDMulWidenHiUint32x4(t *testing.T) {
	var x, y [4]uint32
	for i := range x {
		x[i] = uint32(i*29 - 101)
		y[i] = uint32(77 - i*19)
	}
	a, b := archsimd.LoadUint32x4Array(&x), archsimd.LoadUint32x4Array(&y)
	var got [2]uint64
	a.MulWidenHi(b).StoreArray(&got)
	for i, v := range got {
		if want := uint64(x[i+2]) * uint64(y[i+2]); v != want {
			t.Fatalf("lane %d: got %v want %v", i, v, want)
		}
	}
}
