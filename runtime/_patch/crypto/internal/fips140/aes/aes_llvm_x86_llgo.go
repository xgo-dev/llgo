//go:build llgo && (amd64 || 386) && !baremetal && go1.27 && !go1.28

package aes

import "internal/cpu"

const llgoAESName = "AES-NI"
const llgoAESPreKey = false

// AES-NI-capable x86 processors also provide the XMM/SSE2 register facility.
func llgoAESHardware() bool { return cpu.X86.HasAES }
