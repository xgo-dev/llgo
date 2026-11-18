//go:build llgo && arm && linux && !baremetal && go1.27 && !go1.28

package aes

import _ "unsafe"

const llgoAESName = "ARMv8 Crypto"
const llgoAESPreKey = true

// Linux ARM HWCAP_NEON and HWCAP2_AES describe user-space instruction support.
func llgoAESHardware() bool {
	return llgoAESGetauxval(16)&(1<<12) != 0 && llgoAESGetauxval(26)&1 != 0
}

//go:linkname llgoAESGetauxval C.getauxval
func llgoAESGetauxval(uintptr) uintptr
