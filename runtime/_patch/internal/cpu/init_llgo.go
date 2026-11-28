//go:build (amd64 || 386 || arm64) && !baremetal

package cpu

import _ "unsafe"

// The standard runtime calls Initialize before package initialization. LLGo's
// hosted runtime uses the normal import graph: initialize this package before
// any consumer can read its feature flags, including archsimd and user inits.
// Go's Wasm detector has no options and its early runtime setup does not read
// GODEBUG from the host. Wasm needs no initialization hook here.
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
	llgoCPUFeatures = features | llgoArchSIMDFeatures()
}

//go:linkname llgoCPUFeatures github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures
var llgoCPUFeatures uint64

//go:linkname llgoCPUEnvironment github.com/xgo-dev/llgo/runtime/internal/runtime.CPUEnvironment
func llgoCPUEnvironment() string
