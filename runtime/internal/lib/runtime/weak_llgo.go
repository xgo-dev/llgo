package runtime

import (
	"unsafe"

	latomic "sync/atomic"
	_ "unsafe"

	llrt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// weakHandle supplies the GC-dependent identity used by GOROOT's weak package.
// Go's collector removes weak registrations from spans during sweep. BDWGC
// invokes our cleanup during allocation, so registry removal must be deferred.
// Removing an entry releases the runtime's reference; a user-held weak.Pointer
// must keep its dead handle alive to preserve identity.
type weakHandle struct {
	key  uintptr
	live uint32
	// Written only by the producer before the head CAS publishes this handle;
	// after the head Swap, only the drainer accesses the link. The head atomics
	// order these ordinary accesses. Links retain handles, never referents.
	next unsafe.Pointer
}

// BDWGC conservatively treats pointer-looking uintptr values as live roots.
// Keep weak-pointer identities encoded everywhere they outlive the call that
// registers them, including map keys and weak handles.
func encodeWeakPointer(p unsafe.Pointer) uintptr {
	return ^uintptr(p)
}

func decodeWeakPointer(key uintptr) unsafe.Pointer {
	return unsafe.Pointer(^key)
}

var weakState struct {
	once psync.Once
	mu   psync.Mutex
	m    map[uintptr]*weakHandle
	dead unsafe.Pointer
}

func initWeakState() {
	weakState.mu.Init(nil)
	weakState.m = make(map[uintptr]*weakHandle)
}

// retireWeakHandle runs inside a GC finalizer. Even a map lookup can allocate
// and invoke another finalizer on the same thread, so this path must neither
// allocate nor take weakState.mu. Each handle is published exactly once.
func retireWeakHandle(h *weakHandle) {
	latomic.StoreUint32(&h.live, 0)
	for {
		head := latomic.LoadPointer(&weakState.dead)
		h.next = head
		if latomic.CompareAndSwapPointer(&weakState.dead, head, unsafe.Pointer(h)) {
			return
		}
	}
}

// drainWeakHandles runs during registration with weakState.mu held. Detach only
// one batch so concurrent cleanup cannot keep a registration here indefinitely.
// This bounds the captured work, not its size: a batch of n handles takes O(n)
// time under the registry lock. Without another non-nil weak.Make, the last
// dead batch and its map entries remain retained; neither Value nor GC drains
// them. Reclaiming that idle batch requires a separately scheduled safe consumer.
func drainWeakHandles() {
	for h := (*weakHandle)(latomic.SwapPointer(&weakState.dead, nil)); h != nil; {
		next := (*weakHandle)(h.next)
		// A user-held dead weak.Pointer must not retain the rest of this batch.
		h.next = nil
		// The address may already belong to a new object with a new handle.
		if weakState.m[h.key] == h {
			delete(weakState.m, h.key)
		}
		h = next
	}
}

func llgoRegisterWeakPointer(p unsafe.Pointer) unsafe.Pointer {
	if p == nil {
		return nil
	}
	weakState.once.Do(initWeakState)

	key := encodeWeakPointer(p)
	weakState.mu.Lock()
	drainWeakHandles()
	// A cleanup may mark a handle dead before publishing it to the queue.
	if h := weakState.m[key]; h != nil && latomic.LoadUint32(&h.live) != 0 {
		weakState.mu.Unlock()
		return unsafe.Pointer(h)
	}
	h := &weakHandle{key: key, live: 1}
	weakState.m[key] = h
	weakState.mu.Unlock()

	// Capture only the handle with its encoded identity, never the referent p.
	llrt.AddCleanupPtr(p, func() {
		retireWeakHandle(h)
	})
	return unsafe.Pointer(h)
}

func llgoMakeStrongFromWeak(u unsafe.Pointer) unsafe.Pointer {
	h := (*weakHandle)(u)
	if h == nil || latomic.LoadUint32(&h.live) == 0 {
		return nil
	}
	return decodeWeakPointer(h.key)
}

//go:linkname weak_runtime_registerWeakPointer weak.runtime_registerWeakPointer
func weak_runtime_registerWeakPointer(p unsafe.Pointer) unsafe.Pointer {
	return llgoRegisterWeakPointer(p)
}

//go:linkname weak_runtime_makeStrongFromWeak weak.runtime_makeStrongFromWeak
func weak_runtime_makeStrongFromWeak(u unsafe.Pointer) unsafe.Pointer {
	return llgoMakeStrongFromWeak(u)
}

//go:linkname internal_weak_runtime_registerWeakPointer internal/weak.runtime_registerWeakPointer
func internal_weak_runtime_registerWeakPointer(p unsafe.Pointer) unsafe.Pointer {
	return llgoRegisterWeakPointer(p)
}

//go:linkname internal_weak_runtime_makeStrongFromWeak internal/weak.runtime_makeStrongFromWeak
func internal_weak_runtime_makeStrongFromWeak(u unsafe.Pointer) unsafe.Pointer {
	return llgoMakeStrongFromWeak(u)
}
