//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package vectorcall

import "simd/archsimd"

//go:noinline
func ExtendBytes(x archsimd.Uint8x16) archsimd.Uint16x8 {
	return x.ExtendLo8ToUint16()
}
