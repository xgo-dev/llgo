//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname simdAverage simd/archsimd.Uint8x16.Average
func simdAverage(x, y archsimd.Uint8x16) archsimd.Uint8x16

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Uint8x16
	method := x.Average
	for _, tc := range []struct {
		name   string
		call   func()
		symbol string
	}{
		{"direct", func() { x.Average(x) }, "simd/archsimd.Uint8x16.Average"},
		{"method value", func() { method(x) }, "simd/archsimd.Uint8x16.Average"},
		{"method expression", func() { indirectAverage(archsimd.Uint8x16.Average, x, x) }, "simd/archsimd.Uint8x16.Average"},
		{"deferred", func() { defer x.Average(x) }, "simd/archsimd.Uint8x16.Average"},
		{"linkname", func() { simdAverage(x, x) }, "simd/archsimd.Uint8x16.Average"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				err, ok := recover().(interface{ Error() string })
				if !ok || !strings.HasPrefix(err.Error(), "runtime error: unimplemented SIMD intrinsic: "+tc.symbol) {
					t.Fatalf("unexpected panic: %v", err)
				}
			}()
			tc.call()
			t.Fatal("unimplemented intrinsic returned")
		})
	}
	// Go helper bodies remain executable; the fallback applies to declarations.
	if x.Len() != 16 {
		t.Fatal("Go helper was replaced by the fallback")
	}
}

//go:noinline
func indirectAverage(f func(archsimd.Uint8x16, archsimd.Uint8x16) archsimd.Uint8x16, x, y archsimd.Uint8x16) archsimd.Uint8x16 {
	return f(x, y)
}
