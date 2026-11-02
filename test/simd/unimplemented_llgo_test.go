//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"strings"
	"testing"
	_ "unsafe"
)

//go:linkname simdOnesCount simd/archsimd.Int8x16.OnesCount
func simdOnesCount(x archsimd.Int8x16) archsimd.Int8x16

func TestUnimplementedSIMD(t *testing.T) {
	var x archsimd.Int8x16
	method := x.OnesCount
	for _, tc := range []struct {
		name   string
		call   func()
		symbol string
	}{
		{"direct", func() { x.OnesCount() }, "simd/archsimd.Int8x16.OnesCount"},
		{"method value", func() { method() }, "simd/archsimd.Int8x16.OnesCount"},
		{"method expression", func() { indirectOnesCount(archsimd.Int8x16.OnesCount, x) }, "simd/archsimd.Int8x16.OnesCount"},
		{"deferred", func() { defer x.OnesCount() }, "simd/archsimd.Int8x16.OnesCount"},
		{"linkname", func() { simdOnesCount(x) }, "simd/archsimd.Int8x16.OnesCount"},
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
func indirectOnesCount(f func(archsimd.Int8x16) archsimd.Int8x16, x archsimd.Int8x16) archsimd.Int8x16 {
	return f(x)
}
