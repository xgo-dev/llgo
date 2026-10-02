//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname simdMin simd/archsimd.Float32x4.Min
func simdMin(x, y archsimd.Float32x4) archsimd.Float32x4

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Float32x4
	method := x.Min
	for _, tc := range []struct {
		name   string
		call   func()
		symbol string
	}{
		{"direct", func() { x.Min(x) }, "simd/archsimd.Float32x4.Min"},
		{"method value", func() { method(x) }, "simd/archsimd.Float32x4.Min"},
		{"method expression", func() { indirect(archsimd.Float32x4.Min, x, x) }, "simd/archsimd.Float32x4.Min"},
		{"deferred", func() { defer x.Min(x) }, "simd/archsimd.Float32x4.Min"},
		{"linkname", func() { simdMin(x, x) }, "simd/archsimd.Float32x4.Min"},
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
