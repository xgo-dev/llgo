//go:build darwin || linux || windows || (llgo && wasm)

package runtime

import (
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"unsafe"
)

// Go 1.27 private hooks; scheduler transitions live in the low runtime.

//go:linkname synctest_run internal/synctest.Run
func synctest_run(f func()) {
	rt.SynctestRun(f)
}

//go:linkname synctest_wait internal/synctest.Wait
func synctest_wait() { rt.SynctestWait() }

//go:linkname synctest_isInBubble internal/synctest.IsInBubble
func synctest_isInBubble() bool {
	return rt.CurrentSynctestBubble() != nil
}

//go:linkname synctest_associate internal/synctest.associate
func synctest_associate(p unsafe.Pointer) int {
	return rt.SynctestAssociate(p)
}

//go:linkname synctest_disassociate internal/synctest.disassociate
func synctest_disassociate(p unsafe.Pointer) { rt.SynctestDisassociate(p) }

//go:linkname synctest_isAssociated internal/synctest.isAssociated
func synctest_isAssociated(p unsafe.Pointer) bool {
	return rt.SynctestIsAssociated(p)
}

//go:linkname synctest_acquire internal/synctest.acquire
func synctest_acquire() any {
	b := rt.SynctestAcquire()
	if b == nil {
		return nil
	}
	return b
}

//go:linkname synctest_release internal/synctest.release
func synctest_release(bubble any) {
	if bubble != nil {
		rt.SynctestRelease(bubble.(*rt.SynctestBubble))
	}
}

//go:linkname synctest_inBubble internal/synctest.inBubble
func synctest_inBubble(bubble any, f func()) {
	if bubble == nil {
		f()
		return
	}
	rt.SynctestInBubble(bubble.(*rt.SynctestBubble), f)
}
