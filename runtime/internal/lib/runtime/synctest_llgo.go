//go:build darwin || linux || (llgo && wasm)

package runtime

import "unsafe"

// Minimal synctest stubs for llgo.

//go:linkname synctest_run internal/synctest.Run
func synctest_run(f func()) {
	f()
}

//go:linkname synctest_wait internal/synctest.Wait
func synctest_wait() {}

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

//go:linkname synctest_acquire internal/synctest.acquire
func synctest_acquire() any {
	return nil
}

//go:linkname synctest_release internal/synctest.release
func synctest_release(bubble any) {
}

//go:linkname synctest_inBubble internal/synctest.inBubble
func synctest_inBubble(bubble any, f func()) {
	f()
}
