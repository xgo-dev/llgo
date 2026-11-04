//go:build llgo && arm && !linux && !baremetal && go1.27 && !go1.28

package aes

const llgoAESName = "ARMv8 Crypto"
const llgoAESPreKey = true

func llgoAESHardware() bool { return false }
