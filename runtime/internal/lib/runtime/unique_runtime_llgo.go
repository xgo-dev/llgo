//go:build darwin || linux || windows || (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)

package runtime

import (
	_ "unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/sync"
)

var (
	uniqueMapCleanup     chan struct{}
	uniqueMapCleanupOnce sync.Once
)

//go:linkname unique_runtime_registerUniqueMapCleanup unique.runtime_registerUniqueMapCleanup
func unique_runtime_registerUniqueMapCleanup(f func()) {
	rt.SynctestOutside(func() {
		uniqueMapCleanupOnce.Do(func() {
			uniqueMapCleanup = make(chan struct{}, 1)
		})
		// Start the goroutine in the runtime so it's counted as a system goroutine.
		go func(cleanup func()) {
			markSystemGoroutine()
			for {
				<-uniqueMapCleanup
				cleanup()
			}
		}(f)
	})
}

func unique_runtime_notifyMapCleanup() {
	if uniqueMapCleanup == nil {
		return
	}
	select {
	case uniqueMapCleanup <- struct{}{}:
	default:
	}
}
