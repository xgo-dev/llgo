//go:build (amd64 || 386 || arm64 || wasm) && !baremetal

package cpu

import _ "unsafe"

// The standard runtime calls Initialize before package initialization. LLGo's
// hosted runtime uses the normal import graph: initialize this package before
// any consumer can read its feature flags, including archsimd and user inits.
func init() {
	llgoPrepareCPU()
	Initialize(llgoCPUEnvironment())
	// Publish once, after the official detector has applied GODEBUG. Bit zero
	// distinguishes initialization from an initialized CPU with AVX2 disabled.
	// Keep this layout in sync with internal/llvmfmv's feature bits.
	features := uint64(1)
	if X86.HasAVX2 {
		features |= 2
	}
	if X86.HasAVX {
		features |= 4
	}
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
	llgoCPUFeatures = features
}

//go:linkname llgoCPUFeatures github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures
var llgoCPUFeatures uint64

//go:linkname llgoCPUEnvironment github.com/xgo-dev/llgo/runtime/internal/runtime.CPUEnvironment
func llgoCPUEnvironment() string
