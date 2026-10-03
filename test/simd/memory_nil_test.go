//go:build goexperiment.simd && (amd64 || arm64 || (llgo && wasm))

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// Official Go 1.27's Wasm SIMD array intrinsics access linear-memory address
// zero without a nil check. Keep LLGo's nil-panic contract covered on all
// targets, and retain the official native comparison where it agrees.
func TestSIMDNilArrayPanics(t *testing.T) {
	var v archsimd.Float32x4
	for _, tc := range []struct {
		name string
		f    func()
	}{
		{"load", func() { memoryResult = archsimd.LoadFloat32x4Array(nil) }},
		{"store", func() { v.StoreArray(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing nil array panic")
				}
			}()
			tc.f()
		})
	}
}
