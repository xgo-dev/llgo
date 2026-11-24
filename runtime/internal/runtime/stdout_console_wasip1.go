//go:build llgo && wasm && llgo.wasi_threads

package runtime

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync"
)

// WASI libc's PTHREAD_MUTEX_INITIALIZER is zero. The runtime mutex
// publishes roots while waiting when the linear-memory collector is enabled.
var stdoutDescriptorLock sync.Mutex

//go:linkname lockStdoutDescriptor os.llgoStdoutLock
func lockStdoutDescriptor() { stdoutDescriptorLock.Lock() }

//go:linkname unlockStdoutDescriptor os.llgoStdoutUnlock
func unlockStdoutDescriptor() { stdoutDescriptorLock.Unlock() }
