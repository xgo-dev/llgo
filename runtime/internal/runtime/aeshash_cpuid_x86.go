//go:build llgo && (amd64 || 386) && !baremetal

package runtime

import _ "unsafe"

const aesCPULLGoFiles = "; _wrap/cpuid_x86.c"

//go:linkname llgoCPUHasAES C.llgo_cpu_has_aes
func llgoCPUHasAES() int32

func aesHashHardwareSupport() bool {
	return llgoCPUHasAES() != 0
}
