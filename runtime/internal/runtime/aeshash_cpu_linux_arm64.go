//go:build llgo && linux && arm64 && !baremetal

package runtime

import _ "unsafe"

//go:linkname llgoAESGetauxval C.getauxval
func llgoAESGetauxval(uintptr) uintptr

func aesHashHardwareSupport() bool {
	const (
		atHWCAP  = 16
		hwcapAES = 1 << 3
	)
	return llgoAESGetauxval(atHWCAP)&hwcapAES != 0
}
