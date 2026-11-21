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
	if X86.HasAVXVNNI {
		features |= 1 << 16
	}
	if X86.HasAVX512VPOPCNTDQ {
		features |= 1 << 15
	}
	if X86.HasAVX512VPCLMULQDQ {
		features |= 1 << 14
	}
	if X86.HasAVX512VNNI {
		features |= 1 << 13
	}
	if X86.HasAVX512VBMI2 {
		features |= 1 << 12
	}
	if X86.HasAVX512VBMI {
		features |= 1 << 11
	}
	if X86.HasAVX512BITALG {
		features |= 1 << 10
	}
	if X86.HasFMA {
		features |= 1 << 9
	}
	return features
}
