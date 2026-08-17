//go:build darwin || linux

package runtime

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
)

var (
	uniqueMapCleanup     chan struct{}
	uniqueMapCleanupOnce sync.Once
)

//go:linkname unique_runtime_registerUniqueMapCleanup unique.runtime_registerUniqueMapCleanup
func unique_runtime_registerUniqueMapCleanup(f func()) {
	uniqueMapCleanupOnce.Do(func() {
		uniqueMapCleanup = make(chan struct{}, 1)
	})
	// Start the goroutine in the runtime so it's counted as a system goroutine.
	go func(cleanup func()) {
		for {
			<-uniqueMapCleanup
			cleanup()
		}
	}(f)
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
