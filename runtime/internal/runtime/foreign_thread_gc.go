//go:build llgo && !nogc && !baremetal && (!wasm || (wasip1 && llgo.wasi_threads && !llgo.wasm.gc.linear))

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	"github.com/xgo-dev/llgo/runtime/internal/thread"
)

// EnableForeignThreadRegistration prepares the selected collector for threads
// created outside the LLGo runtime. Library-mode initialization calls it from
// an already registered thread before any C export can run.
func EnableForeignThreadRegistration() {
	bdwgc.Init()
	bdwgc.AllowRegisterThreads()
}

// gLifecycleDestructor selects a wrapper for GC builds. Collector teardown
// must happen when the host-thread lifecycle ends, not at every destroyG call.
func gLifecycleDestructor(_ thread.KeyDestructor) thread.KeyDestructor {
	return destroyForeignG
}

func destroyForeignG(ptr c.Pointer) {
	gp := (*g)(ptr)
	// Capture collector ownership before destroyG releases context.root and
	// invalidates gp.context.
	attached := gp != nil && gp.context != nil && gp.context.platform.foreignThreadAttached
	destroyG(ptr)
	// This is the last Go operation in the TLS destructor. A foreign thread
	// that entered through a C boundary can now leave the collector safely.
	if attached {
		bdwgc.UnregisterMyThread()
	}
}

// EnterForeignThread makes a thread created outside the LLGo runtime visible
// to the collector before it allocates or manipulates Go pointers. The return
// value records whether this call installed the retained registration.
func EnterForeignThread() bool {
	gp := (*g)(unsafe.Pointer(currentG))
	// Runtime-created threads are registered by the collector-aware thread
	// creation API. Their G has no TLS lifecycle because mexit owns teardown.
	if gp != nil {
		if gp.context != nil && gp.context.platform.foreignThreadAttached {
			return false
		}
		if !currentGHasLifecycle {
			return false
		}
	}
	if bdwgc.ThreadIsRegistered() != 0 {
		return false
	}
	// An address-taken Go local may escape to GC_malloc. This thread is not
	// registered yet, so keep the collector's stack descriptor on the native
	// stack until registration completes.
	base := (*bdwgc.StackBase)(c.Alloca(unsafe.Sizeof(bdwgc.StackBase{})))
	if bdwgc.GetStackBase(base) != bdwgc.Success {
		// A Go panic is not safe until the collector knows this thread.
		c.Fputs(c.Str("runtime: failed to discover foreign thread stack\n"), c.Stderr)
		c.Exit(2)
		return false
	}
	switch status := bdwgc.RegisterMyThread(base); status {
	case bdwgc.Success:
		return retainForeignThreadRegistration()
	case bdwgc.Duplicate:
		return false
	default:
		c.Fputs(c.Str("runtime: failed to register foreign thread\n"), c.Stderr)
		c.Exit(2)
		return false
	}
}

// retainForeignThreadRegistration runs only after the collector knows about
// the current thread. Keep its defer out of EnterForeignThread: setting up a
// Go defer needs getg and may allocate, which is not safe before registration.
func retainForeignThreadRegistration() bool {
	ready := false
	defer func() {
		if !ready {
			bdwgc.UnregisterMyThread()
		}
	}()
	// Ensure the existing G lifecycle key will release the manual GC
	// registration when this host thread exits.
	gp := getg()
	gp.context.platform.foreignThreadAttached = true
	ready = true
	return true
}

// ExitForeignThread retains an LLGo-owned registration when the current G has
// a TLS lifecycle. The lifecycle destructor releases it after its last Go/GC
// operation. The fallback covers an entry that could not install a lifecycle.
func ExitForeignThread(registered bool) {
	if registered && !currentGUsesLifecycle() {
		bdwgc.UnregisterMyThread()
	}
}
