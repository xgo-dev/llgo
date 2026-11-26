package main

import "simd/archsimd"

type Data struct {
	F   [4]float32
	U4  [4]uint32
	U8  [8]uint32
	U16 [16]uint32
	B16 [16]uint8
	B32 [32]uint8
	B64 [64]uint8
	M   uint64
}

//go:noinline
func kernel(p *Data) {
	if !archsimd.X86.AVX() {
		return
	}
	if archsimd.X86.FMA() {
		x := archsimd.LoadFloat32x4Array(&p.F)
		x.MulAdd(x, x).StoreArray(&p.F)
	}
	if archsimd.X86.SHA() {
		x := archsimd.LoadUint32x4Array(&p.U4)
		x.SHA256TwoRounds(x, x).StoreArray(&p.U4)
	}
	if archsimd.X86.AVXAES() {
		archsimd.LoadUint8x16Array(&p.B16).AESEncryptOneRound(archsimd.LoadUint32x4Array(&p.U4)).StoreArray(&p.B16)
	}
	if archsimd.X86.VAES() {
		archsimd.LoadUint8x32Array(&p.B32).AESEncryptOneRound(archsimd.LoadUint32x8Array(&p.U8)).StoreArray(&p.B32)
	}
	if archsimd.X86.AVX512VAES() {
		archsimd.LoadUint8x64Array(&p.B64).AESEncryptOneRound(archsimd.LoadUint32x16Array(&p.U16)).StoreArray(&p.B64)
	}
	if archsimd.X86.AVX512GFNI() {
		x := archsimd.LoadUint8x64Array(&p.B64)
		x.GaloisFieldMul(x).StoreArray(&p.B64)
	}
	if archsimd.X86.AVX512BITALG() {
		archsimd.LoadUint8x64Array(&p.B64).OnesCount().StoreArray(&p.B64)
	}
	if archsimd.X86.AVX512VBMI2() {
		archsimd.LoadUint8x64Array(&p.B64).Compress(archsimd.Mask8x64FromBits(p.M)).StoreArray(&p.B64)
	}
	if archsimd.X86.AVX512VPOPCNTDQ() {
		archsimd.LoadUint32x16Array(&p.U16).OnesCount().StoreArray(&p.U16)
	}
}
func main() { p := Data{M: 0x555555}; kernel(&p); println(p.F[0], p.U16[0], p.B64[0]) }
