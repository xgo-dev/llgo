//go:build (amd64 || 386 || arm64 || wasm) && !baremetal && go1.27

package cpu

func llgoArchSIMDFeatures() uint64 {
	var features uint64
	// Match the effective predicates exported by simd/archsimd on AMD64.
	if X86.HasAES && X86.HasAVX {
		features |= 8
	}
	if X86.HasVAES {
		features |= 16
	}
	if X86.HasAVX512VAES {
		features |= 32
	}
	if X86.HasAVX512GFNI {
		features |= 64
	}
	if X86.HasSHA {
		features |= 128
	}
	if X86.HasAVX512 {
		features |= 256
	}
	return features
}
