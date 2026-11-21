//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	_ "unsafe"
)

//go:linkname unsupportedEntry simd/archsimd.Int8x16.MulSign
func unsupportedEntry(x archsimd.Int8x16, y archsimd.Int8x16) archsimd.Int8x16

func unsupportedCalls(x archsimd.Int8x16) (string, []unsupportedCall) {
	method := x.MulSign
	return "simd/archsimd.Int8x16.MulSign", []unsupportedCall{
		{"direct", func() { x.MulSign(x) }},
		{"method value", func() { method(x) }},
		{"method expression", func() { invokeUnsupportedPair(archsimd.Int8x16.MulSign, x) }},
		{"deferred", func() { defer x.MulSign(x) }},
		{"linkname", func() { unsupportedEntry(x, x) }},
	}
}
