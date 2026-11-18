//go:build goexperiment.simd && amd64

package main

import "simd/archsimd"

//go:noinline
func recoverWide(x, y archsimd.Float32x8) (result archsimd.Float32x8) {
	defer func() {
		if recover() == nil {
			panic("missing recovery")
		}
		result = x.Sub(y)
	}()
	panic("wide value remains live")
}

func main() {
	if !archsimd.X86.AVX2() {
		panic("this official Go reproducer requires AVX2")
	}
	x, y := archsimd.BroadcastFloat32x8(9), archsimd.BroadcastFloat32x8(2)
	var lanes [8]float32
	recoverWide(x, y).StoreArray(&lanes)
	for _, v := range lanes {
		if v != 7 {
			panic("SIMD256 recover result lost upper lanes")
		}
	}
}
