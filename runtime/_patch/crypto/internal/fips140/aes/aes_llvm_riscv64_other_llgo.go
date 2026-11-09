//go:build llgo && riscv64 && !linux && !baremetal && go1.27 && !go1.28

package aes

const llgoAESName = "RISC-V Crypto"
const llgoAESPreKey = false

func llgoAESHardware() bool { return false }
