//go:build llgo && goexperiment.simd && amd64

package simd_test

import (
	"simd/archsimd"
	_ "unsafe"
)

//go:linkname unsupportedEntry simd/archsimd.Uint32x4.SHA1Message1
func unsupportedEntry(x archsimd.Uint32x4, y archsimd.Uint32x4) archsimd.Uint32x4

func unsupportedCalls(x archsimd.Uint32x4) (string, []unsupportedCall) {
	method := x.SHA1Message1
	return "simd/archsimd.Uint32x4.SHA1Message1", []unsupportedCall{
		{"direct", func() { x.SHA1Message1(x) }},
		{"method value", func() { method(x) }},
		{"method expression", func() { invokeUnsupportedPair(archsimd.Uint32x4.SHA1Message1, x) }},
		{"deferred", func() { defer x.SHA1Message1(x) }},
		{"linkname", func() { unsupportedEntry(x, x) }},
	}
}
