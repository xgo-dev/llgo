//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname simdDiv simd/archsimd.Float32x4.Div
func simdDiv(x, y archsimd.Float32x4) archsimd.Float32x4

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Float32x4
	method := x.Div
	for _, tc := range []struct {
		name   string
		call   func()
		symbol string
	}{
		{"direct", func() { x.Div(x) }, "simd/archsimd.Float32x4.Div"},
		{"method value", func() { method(x) }, "simd/archsimd.Float32x4.Div"},
		{"method expression", func() { indirect(archsimd.Float32x4.Div, x, x) }, "simd/archsimd.Float32x4.Div"},
		{"deferred", func() { defer x.Div(x) }, "simd/archsimd.Float32x4.Div"},
		{"linkname", func() { simdDiv(x, x) }, "simd/archsimd.Float32x4.Div"},
		{"mask result", func() { x.Equal(x) }, "simd/archsimd.Float32x4.Equal"},
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
	if x.Len() != 4 {
		t.Fatal("Go helper was replaced by the fallback")
	}
}
