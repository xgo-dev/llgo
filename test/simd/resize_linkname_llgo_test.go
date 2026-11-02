//go:build llgo && goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"testing"
	_ "unsafe"
)

//go:linkname resizeExtend simd/archsimd.Uint8x16.ExtendLo8ToUint16
func resizeExtend(archsimd.Uint8x16) archsimd.Uint16x8

func TestSIMDResizeLinkname(t *testing.T) {
	testSIMDIntegerResize(t, "lo", func(in []uint8, out []uint16) {
		resizeExtend(archsimd.LoadUint8x16(in)).Store(out)
	})
}
