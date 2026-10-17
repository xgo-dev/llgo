//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package simd_test

import (
	"simd/archsimd"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/internal/vectorcall"
)

func TestSIMDResizeCallBoundary(t *testing.T) {
	input := [16]uint8{0, 127, 128, 255, 1, 2, 3, 4, 99, 98, 97, 96, 95, 94, 93, 92}
	x := archsimd.LoadUint8x16Array(&input)
	method := x.ExtendLo8ToUint16
	for _, got := range []archsimd.Uint16x8{
		method(),
		invokeResize(archsimd.Uint8x16.ExtendLo8ToUint16, x),
		vectorcall.ExtendBytes(x),
	} {
		var values [8]uint16
		got.StoreArray(&values)
		for i, v := range values {
			if v != uint16(input[i]) {
				t.Fatalf("lane %d: %d, want %d", i, v, input[i])
			}
		}
	}
	defer x.ExtendLo8ToUint16()
}

//go:noinline
func invokeResize(f func(archsimd.Uint8x16) archsimd.Uint16x8, x archsimd.Uint8x16) archsimd.Uint16x8 {
	return f(x)
}
