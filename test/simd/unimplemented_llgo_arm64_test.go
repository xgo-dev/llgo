//go:build llgo && goexperiment.simd && arm64

package simd_test

import (
	"simd/archsimd"
	_ "unsafe"
)

//go:linkname unsupportedEntry simd/archsimd.Int8x16.LeadingZeros
func unsupportedEntry(x archsimd.Int8x16) archsimd.Int8x16

func unsupportedCalls(x archsimd.Int8x16) (string, []unsupportedCall) {
	method := x.LeadingZeros
	return "simd/archsimd.Int8x16.LeadingZeros", []unsupportedCall{
		{"direct", func() { x.LeadingZeros() }},
		{"method value", func() { method() }},
		{"method expression", func() { invokeUnsupported(archsimd.Int8x16.LeadingZeros, x) }},
		{"deferred", func() { defer x.LeadingZeros() }},
		{"linkname", func() { unsupportedEntry(x) }},
	}
}
