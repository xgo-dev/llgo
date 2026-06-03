//go:build darwin || linux

package runtime

import "unsafe"

// Minimal synctest stubs for llgo.

//go:linkname synctest_isInBubble internal/synctest.IsInBubble
func synctest_isInBubble() bool {
	return false
}

//go:linkname synctest_associate internal/synctest.associate
func synctest_associate(_ unsafe.Pointer) int {
	return 0
}

//go:linkname synctest_disassociate internal/synctest.disassociate
func synctest_disassociate(_ unsafe.Pointer) {}

//go:linkname synctest_isAssociated internal/synctest.isAssociated
func synctest_isAssociated(_ unsafe.Pointer) bool {
	return false
}
