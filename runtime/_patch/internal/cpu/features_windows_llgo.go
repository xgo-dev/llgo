//go:build windows && !baremetal

package cpu

import _ "unsafe"

// Preserve the Go bool / Win32 BOOL boundary without a public-runtime import.
func isProcessorFeaturePresent(feature uint32) bool {
	return llgoProcessorFeaturePresent(feature) != 0
}

//go:linkname llgoProcessorFeaturePresent stdcall.IsProcessorFeaturePresent
func llgoProcessorFeaturePresent(feature uint32) int32
