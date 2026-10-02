//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname simdConvert simd/archsimd.Float32x4.ConvertToInt32
func simdConvert(x archsimd.Float32x4) archsimd.Int32x4

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Float32x4
	method := x.ConvertToInt32
	for _, tc := range []struct {
		name   string
		call   func()
		symbol string
	}{
		{"direct", func() { x.ConvertToInt32() }, "simd/archsimd.Float32x4.ConvertToInt32"},
		{"method value", func() { method() }, "simd/archsimd.Float32x4.ConvertToInt32"},
		{"method expression", func() { indirectConvert(archsimd.Float32x4.ConvertToInt32, x) }, "simd/archsimd.Float32x4.ConvertToInt32"},
		{"deferred", func() { defer x.ConvertToInt32() }, "simd/archsimd.Float32x4.ConvertToInt32"},
		{"linkname", func() { simdConvert(x) }, "simd/archsimd.Float32x4.ConvertToInt32"},
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

//go:noinline
func indirectConvert(f func(archsimd.Float32x4) archsimd.Int32x4, x archsimd.Float32x4) archsimd.Int32x4 {
	return f(x)
}
