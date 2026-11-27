package main

import "simd/archsimd"

//go:noinline
func shaKernel(p *[4]uint32) {
	if archsimd.X86.SHA() {
		x := archsimd.LoadUint32x4Array(p)
		x.SHA256TwoRounds(x, x).StoreArray(p)
	}
}

//go:noinline
func fmaKernel(p *[4]float32) {
	if archsimd.X86.FMA() {
		x := archsimd.LoadFloat32x4Array(p)
		x.MulAdd(x, x).StoreArray(p)
	}
}

func main() {
	shaKernel(new([4]uint32))
	fmaKernel(new([4]float32))
}
