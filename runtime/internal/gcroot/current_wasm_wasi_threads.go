//go:build llgo && wasip1 && wasm && llgo.wasm.gc.linear && llgo.wasi_threads

package gcroot

import "unsafe"

// The compiler names these slots directly. Each WASI pthread must keep its
// own root chain, including the main thread and threads entering from C.
// Address-bearing slots use uintptr so the root-chain boundary helpers do not
// acquire compiler root frames while changing the chain itself.
//
//llgointernal:tls
var (
	currentRootChain uintptr
	sjljReplaying    bool
	activeContext    uintptr
	rebuilding       bool
)

func currentThreadContext() *Context {
	return (*Context)(unsafe.Pointer(gcrootThreadContext()))
}

//go:linkname gcrootThreadContext C.llgo_gcroot_thread_context
func gcrootThreadContext() uintptr

func CurrentThreadRegistered() bool { return activeContext != 0 }

// RegisterCurrentThread must run inside the WASI world's registration gate,
// before this thread makes any allocation that can collect.
func RegisterCurrentThread() {
	ctx := currentThreadContext()
	if activeContext != 0 {
		return
	}
	Register(ctx)
	activeContext = uintptr(unsafe.Pointer(ctx))
}

// UnregisterCurrentThread must run inside the world's registration gate, after
// the thread has released its last Go-owned object.
func UnregisterCurrentThread() {
	ctx := currentThreadContext()
	if activeContext == 0 {
		return
	}
	ctx.chain = nil
	activeContext = 0
	ctx.stackBottom = 0
	ctx.stackTop = 0
	Unregister(ctx)
}

// VisitThreadStacks is called only after every registered mutator has stopped.
// The collector scans its own current stack separately.
func VisitThreadStacks(visitor func(bottom, top uintptr)) {
	lockRegistry()
	owner := (*Context)(unsafe.Pointer(activeContext))
	for ctx := contexts; ctx != nil; ctx = ctx.next {
		if ctx == owner {
			continue
		}
		if ctx.stackBottom == 0 || ctx.stackBottom >= ctx.stackTop {
			unlockRegistry()
			panic("gcroot: stopped WASI pthread has no stack range")
		}
		visitor(ctx.stackBottom, ctx.stackTop)
	}
	unlockRegistry()
}
