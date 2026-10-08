//go:build llgo && wasm && !baremetal && llgo.wasm.debugger

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

const wasmDebuggerEnabled = true

// wasmDebuggerNode describes a logical G, including a parked fiber. The
// debugger must stop all workers before reading mutable G and caller state.
// Nodes use libc storage so registering the initial G cannot recurse into
// Go allocation before its local context has been installed.
type wasmDebuggerNode struct {
	next, previous *wasmDebuggerNode
	gp             *g
	callers        *callerLocationStore
	sequence       uint32
	updateDepth    uint32
	processorID    int32 // last P; -1 until first execution
}

var wasmDebuggerRegistry struct {
	version uint32
	epoch   uint32
	head    *wasmDebuggerNode
}

var wasmDebuggerLock uint32
var TracebackCreator func() uintptr

func lockWasmDebugger() {
	for {
		if _, ok := atomic.CompareAndExchange(&wasmDebuggerLock, uint32(0), uint32(1)); ok {
			return
		}
		CooperativeSafepoint()
	}
}

// The shared parent parameter is unused: gp.parentGoid supplies the debugger linkage.
func registerTraceback(gp, _ *g) {
	node := (*wasmDebuggerNode)(c.Malloc(unsafe.Sizeof(wasmDebuggerNode{})))
	if node == nil {
		fatal("runtime: failed to allocate debugger goroutine record")
		return
	}
	*node = wasmDebuggerNode{gp: gp, processorID: -1}
	lockWasmDebugger()
	atomic.Add(&wasmDebuggerRegistry.epoch, uint32(1))
	// Initialization precedes Go package init; avoid an initializer that
	// would subsequently overwrite the already-published registry head.
	wasmDebuggerRegistry.version = 1
	node.next = wasmDebuggerRegistry.head
	if node.next != nil {
		node.next.previous = node
	}
	wasmDebuggerRegistry.head = node
	gp.tracebackThread = unsafe.Pointer(node)
	atomic.Add(&wasmDebuggerRegistry.epoch, uint32(1))
	atomic.Store(&wasmDebuggerLock, uint32(0))
}

func unregisterTraceback(gp *g) {
	node := (*wasmDebuggerNode)(gp.tracebackThread)
	if node == nil {
		return
	}
	lockWasmDebugger()
	atomic.Add(&wasmDebuggerRegistry.epoch, uint32(1))
	if node.previous == nil {
		wasmDebuggerRegistry.head = node.next
	} else {
		node.previous.next = node.next
	}
	if node.next != nil {
		node.next.previous = node.previous
	}
	gp.tracebackThread = nil
	atomic.Add(&wasmDebuggerRegistry.epoch, uint32(1))
	atomic.Store(&wasmDebuggerLock, uint32(0))
	c.Free(unsafe.Pointer(node))
}

func publishWasmCallerStore(store *callerLocationStore) {
	if gp := getg(); gp != nil {
		if node := (*wasmDebuggerNode)(gp.tracebackThread); node != nil {
			node.callers = store
		}
	}
}

func beginWasmCallerUpdate() unsafe.Pointer {
	gp := getg()
	if gp == nil || gp.tracebackThread == nil {
		return nil
	}
	node := (*wasmDebuggerNode)(gp.tracebackThread)
	if node.updateDepth == 0 {
		atomic.Add(&node.sequence, uint32(1))
	}
	node.updateDepth++
	if gp.m != nil && gp.m.p != nil {
		node.processorID = gp.m.p.id
	}
	return unsafe.Pointer(node)
}

func endWasmCallerUpdate(pointer unsafe.Pointer) {
	if pointer == nil {
		return
	}
	node := (*wasmDebuggerNode)(pointer)
	node.updateDepth--
	if node.updateDepth == 0 {
		atomic.Add(&node.sequence, uint32(1))
	}
}

func attachTraceback(*g)    {}
func releaseFaultSnapshot() {}
func TracebackWaiting(bool) {}
