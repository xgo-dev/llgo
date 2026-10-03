//go:build (amd64 || arm64 || wasm) && !baremetal

package cpu

import _ "unsafe"

// The standard runtime calls Initialize before package initialization. LLGo's
// hosted runtime uses the normal import graph: initialize this package before
// any consumer can read its feature flags, including archsimd and user inits.
func init() {
	llgoPrepareCPU()
	Initialize(llgoCPUEnvironment())
}

//go:linkname llgoCPUEnvironment github.com/xgo-dev/llgo/runtime/internal/runtime.CPUEnvironment
func llgoCPUEnvironment() string
