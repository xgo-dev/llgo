//go:build llgo && js && wasm && llgo.wasm.workers

package js

import (
	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
	"github.com/xgo-dev/llgo/runtime/internal/wasmsync"
)

// Registry and poll-registration sections contain no G suspension. Keep
// Unlock free of the entry safepoint in sync.Mutex.Unlock: preempting a holder
// strands synchronous callbacks behind unrelated JS callers on this worker.
type funcMutex struct{ mutex wasmsync.Mutex }

//go:nosplit
func (m *funcMutex) Lock() { m.mutex.Lock(llruntime.CooperativeSafepoint) }

//go:nosplit
func (m *funcMutex) Unlock() { m.mutex.Unlock() }
