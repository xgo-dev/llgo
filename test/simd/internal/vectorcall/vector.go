//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package vectorcall

import "simd/archsimd"

//go:noinline
func Echo(x archsimd.Float32x4, fail bool) archsimd.Float32x4 {
	if fail {
		panic("vector call")
	}
	return x.Add(archsimd.BroadcastFloat32x4(2))
}

// Scalar intentionally allows inlining. The Emscripten backend must prevent a
// late inliner from moving its vector call into a caller with a recovery point.
func Scalar(x float32, fail bool) float32 {
	return Echo(archsimd.BroadcastFloat32x4(x), fail).GetElem(0)
}
