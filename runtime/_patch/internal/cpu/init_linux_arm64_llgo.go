//go:build linux && arm64 && !baremetal

package cpu

import _ "unsafe"

// Linux supplies the effective user-space CPU capabilities in AT_HWCAP.
// Populate the official variable before its platform detection reads it.
func llgoPrepareCPU() {
	const atHWCAP = 16
	HWCap = uint(llgoGetauxval(atHWCAP))
}

//go:linkname llgoGetauxval C.getauxval
func llgoGetauxval(uintptr) uintptr
