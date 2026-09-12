package runtime

import (
	"unsafe"

	latomic "sync/atomic"
	_ "unsafe"

	llrt "github.com/xgo-dev/llgo/runtime/internal/runtime"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

type weakHandle struct {
	key  uintptr
	live uint32
	next unsafe.Pointer // next dead handle; never a pointer to the referent
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
func drainWeakHandles() {
	for h := (*weakHandle)(latomic.SwapPointer(&weakState.dead, nil)); h != nil; {
		next := (*weakHandle)(h.next)
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
