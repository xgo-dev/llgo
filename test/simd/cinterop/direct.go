//go:build llgo && goexperiment.simd && (amd64 || arm64)

package cinterop

import (
	"simd/archsimd"
	_ "unsafe"
)

const LLGoFiles = "wrap/direct.c"

//llgo:cdecl
type vectorCallback func(archsimd.Float32x4, archsimd.Float32x4) archsimd.Float32x4

//llgo:cdecl
type nativeBox struct {
	tag  byte
	v    archsimd.Float32x4
	tail byte
}

//llgo:cdecl
type nativePair struct{ a, b archsimd.Float32x4 }

//llgo:cdecl
type pairCallback func(nativePair) nativePair

//go:linkname directPairCallback C.llgo_direct_pair_callback
func directPairCallback(fn pairCallback, p nativePair) nativePair

//go:noinline
func swapNativePair(p nativePair) nativePair { return nativePair{p.b, p.a} }

type embeddedTag byte

//llgo:cdecl
type nativeEmbedded struct {
	*embeddedTag
	v    archsimd.Float32x4
	tail byte
}

//go:linkname directAdd C.llgo_direct_add
func directAdd(a, b archsimd.Float32x4) archsimd.Float32x4

//go:linkname directCallback C.llgo_direct_callback
func directCallback(fn vectorCallback, a, b archsimd.Float32x4) archsimd.Float32x4

//go:linkname directExport C.llgo_direct_export
func directExport(a, b archsimd.Float32x4) archsimd.Float32x4

//go:linkname directReadBox C.llgo_direct_box_read
func directReadBox(p *nativeBox) float32

//go:linkname directBox C.llgo_direct_box
func directBox(p nativeBox) nativeBox

//go:linkname directPair C.llgo_direct_pair
func directPair(p nativePair) nativePair

//export llgo_test_vector_export
func llgo_test_vector_export(a, b archsimd.Float32x4) archsimd.Float32x4 { return a.Sub(b) }

//go:noinline
func vectorSubtract(a, b archsimd.Float32x4) archsimd.Float32x4 { return a.Sub(b) }
