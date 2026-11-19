//go:build llgo && goexperiment.simd && wasm

package simd_test

import (
	"simd/archsimd"
	_ "unsafe"
)

//go:linkname unsupportedEntry simd/archsimd.Int8x16.MulWidenHi
func unsupportedEntry(x archsimd.Int8x16, y archsimd.Int8x16) archsimd.Int16x8

func unsupportedCalls(x archsimd.Int8x16) (string, []unsupportedCall) {
	method := x.MulWidenHi
	return "simd/archsimd.Int8x16.MulWidenHi", []unsupportedCall{
		{"direct", func() { x.MulWidenHi(x) }},
		{"method value", func() { method(x) }},
		{"method expression", func() { invokeUnsupportedPair(archsimd.Int8x16.MulWidenHi, x) }},
		{"deferred", func() { defer x.MulWidenHi(x) }},
		{"linkname", func() { unsupportedEntry(x, x) }},
	}
}
