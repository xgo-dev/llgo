//go:build llgo && windows && arm64 && !baremetal

package runtime

import _ "unsafe"

//go:linkname llgoAESProcessorFeaturePresent stdcall.IsProcessorFeaturePresent
func llgoAESProcessorFeaturePresent(feature uint32) int32

func aesHashHardwareSupport() bool {
	const pfARMV8CryptoInstructionsAvailable = 30
	return llgoAESProcessorFeaturePresent(pfARMV8CryptoInstructionsAvailable) != 0
}
