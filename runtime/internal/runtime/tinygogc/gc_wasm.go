//go:build wasm && llgo_wasm_gc

package tinygogc

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/gcroot"
)

const LLGoFiles = "_wrap/gc_wasm.c"

const wasmPageSize = uintptr(64 << 10)

func gcMemoryLayout() (heapStart, heapEnd, globalsStart, globalsEnd, stackTop uintptr) {
	heapStart = alignUp(gcWasmHeapBase(), bytesPerBlock)
	heapEnd = alignDown(gcWasmMemorySize(), bytesPerBlock)
	globalsStart = gcWasmGlobalsStart()
	globalsEnd = gcWasmGlobalsEnd()
	stackTop = gcWasmStackTop()
	return
}

func gcGrowMemory(oldHeapEnd uintptr) uintptr {
	current := gcWasmMemorySize()
	if current < oldHeapEnd {
		return oldHeapEnd
	}
	growth := oldHeapEnd - heapStart
	if growth < 1<<20 {
		growth = 1 << 20
	}
	if current > ^uintptr(0)-growth {
		return oldHeapEnd
	}
	required := alignUp(current+growth, wasmPageSize)
	if gcWasmGrowMemory(required) == 0 {
		return oldHeapEnd
	}
	return alignDown(gcWasmMemorySize(), bytesPerBlock)
}

func gcMarkReachable() {
	sp := uintptr(getsp())
	top := gcWasmStackTop()
	if sp < top {
		markRoots(sp, top)
	}
	if globalsStart < globalsEnd {
		markRoots(globalsStart, globalsEnd)
	}
	gcroot.Visit(markWasmGCRoot)
}

func markWasmGCRoot(root *unsafe.Pointer, _ unsafe.Pointer) {
	markRoot(uintptr(unsafe.Pointer(root)), uintptr(*root))
}

func gcStackStats() (inuse, sys uintptr) {
	sp := uintptr(getsp())
	top := gcWasmStackTop()
	if sp < top {
		inuse = top - sp
		sys = inuse
	}
	return
}

func alignUp(value, alignment uintptr) uintptr {
	return (value + alignment - 1) &^ (alignment - 1)
}

func alignDown(value, alignment uintptr) uintptr {
	return value &^ (alignment - 1)
}

//go:linkname gcWasmGlobalsStart C.llgo_gc_globals_start
func gcWasmGlobalsStart() uintptr

//go:linkname gcWasmGlobalsEnd C.llgo_gc_globals_end
func gcWasmGlobalsEnd() uintptr

//go:linkname gcWasmHeapBase C.llgo_gc_heap_base
func gcWasmHeapBase() uintptr

//go:linkname gcWasmStackTop C.llgo_gc_stack_top
func gcWasmStackTop() uintptr

//go:linkname gcWasmMemorySize C.llgo_gc_memory_size
func gcWasmMemorySize() uintptr

//go:linkname gcWasmGrowMemory C.llgo_gc_grow_memory
func gcWasmGrowMemory(required uintptr) c.Int
