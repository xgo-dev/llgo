//go:build llgo && arm64 && !baremetal && go1.27 && !go1.28

package aes

import "internal/cpu"

const llgoAESName = "Armv8.0"
const llgoAESPreKey = true

func llgoAESHardware() bool { return cpu.ARM64.HasAES }
