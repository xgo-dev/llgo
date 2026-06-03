//go:build baremetal

package tinygogc

import "unsafe"

type GCStats struct {
	// General statistics.

	// Alloc is bytes of allocated heap objects.
	//
	// This is the same as HeapAlloc (see below).
	Alloc uint64

	// TotalAlloc is cumulative bytes allocated for heap objects.
	//
	// TotalAlloc increases as heap objects are allocated, but
	// unlike Alloc and HeapAlloc, it does not decrease when
	// objects are freed.
	TotalAlloc uint64

	// Sys is the total bytes of memory obtained from the OS.
	//
	// Sys is the sum of the XSys fields below. Sys measures the
	// virtual address space reserved by the Go runtime for the
	// heap, stacks, and other internal data structures. It's
	// likely that not all of the virtual address space is backed
	// by physical memory at any given moment, though in general
	// it all was at some point.
	Sys uint64

	// Mallocs is the cumulative count of heap objects allocated.
	// The number of live objects is Mallocs - Frees.
	Mallocs uint64

	// Frees is the cumulative count of heap objects freed.
	Frees uint64

	// Heap memory statistics.
	//
	// Interpreting the heap statistics requires some knowledge of
	// how Go organizes memory. Go divides the virtual address
	// space of the heap into "spans", which are contiguous
	// regions of memory 8K or larger. A span may be in one of
	// three states:
	//
	// An "idle" span contains no objects or other data. The
	// physical memory backing an idle span can be released back
	// to the OS (but the virtual address space never is), or it
	// can be converted into an "in use" or "stack" span.
	//
	// An "in use" span contains at least one heap object and may
	// have free space available to allocate more heap objects.
	//
	// A "stack" span is used for goroutine stacks. Stack spans
	// are not considered part of the heap. A span can change
	// between heap and stack memory; it is never used for both
	// simultaneously.

	// HeapAlloc is bytes of allocated heap objects.
	//
	// "Allocated" heap objects include all reachable objects, as
	// well as unreachable objects that the garbage collector has
	// not yet freed. Specifically, HeapAlloc increases as heap
	// objects are allocated and decreases as the heap is swept
	// and unreachable objects are freed. Sweeping occurs
	// incrementally between GC cycles, so these two processes
	// occur simultaneously, and as a result HeapAlloc tends to
	// change smoothly (in contrast with the sawtooth that is
	// typical of stop-the-world garbage collectors).
	HeapAlloc uint64

	// HeapSys is bytes of heap memory obtained from the OS.
	//
	// HeapSys measures the amount of virtual address space
	// reserved for the heap. This includes virtual address space
	// that has been reserved but not yet used, which consumes no
	// physical memory, but tends to be small, as well as virtual
	// address space for which the physical memory has been
	// returned to the OS after it became unused (see HeapReleased
	// for a measure of the latter).
	//
	// HeapSys estimates the largest size the heap has had.
	HeapSys uint64

	// HeapIdle is bytes in idle (unused) spans.
	//
	// Idle spans have no objects in them. These spans could be
	// (and may already have been) returned to the OS, or they can
	// be reused for heap allocations, or they can be reused as
	// stack memory.
	//
	// HeapIdle minus HeapReleased estimates the amount of memory
	// that could be returned to the OS, but is being retained by
	// the runtime so it can grow the heap without requesting more
	// memory from the OS. If this difference is significantly
	// larger than the heap size, it indicates there was a recent
	// transient spike in live heap size.
	HeapIdle uint64

	// HeapInuse is bytes in in-use spans.
	//
	// In-use spans have at least one object in them. These spans
	// can only be used for other objects of roughly the same
	// size.
	//
	// HeapInuse minus HeapAlloc estimates the amount of memory
	// that has been dedicated to particular size classes, but is
	// not currently being used. This is an upper bound on
	// fragmentation, but in general this memory can be reused
	// efficiently.
	HeapInuse uint64

	// Stack memory statistics.
	//
	// Stacks are not considered part of the heap, but the runtime
	// can reuse a span of heap memory for stack memory, and
	// vice-versa.

	// StackInuse is bytes in stack spans.
	//
	// In-use stack spans have at least one stack in them. These
	// spans can only be used for other stacks of the same size.
	//
	// There is no StackIdle because unused stack spans are
	// returned to the heap (and hence counted toward HeapIdle).
	StackInuse uint64

	// StackSys is bytes of stack memory obtained from the OS.
	//
	// StackSys is StackInuse, plus any memory obtained directly
	// from the OS for OS thread stacks.
	//
	// In non-cgo programs this metric is currently equal to StackInuse
	// (but this should not be relied upon, and the value may change in
	// the future).
	//
	// In cgo programs this metric includes OS thread stacks allocated
	// directly from the OS. Currently, this only accounts for one stack in
	// c-shared and c-archive build modes and other sources of stacks from
	// the OS (notably, any allocated by C code) are not currently measured.
	// Note this too may change in the future.
	StackSys uint64

	// GCSys is bytes of memory in garbage collection metadata.
	GCSys uint64
}

func ReadGCStats() GCStats {
	var heapInuse, heapIdle uint64

	lock(&gcMutex)

	for block := uintptr(0); block < endBlock; block++ {
		bstate := gcStateOf(block)
		if bstate == blockStateFree {
			heapIdle += uint64(bytesPerBlock)
		} else {
			heapInuse += uint64(bytesPerBlock)
		}
	}

	stackEnd := uintptr(unsafe.Pointer(&_stackEnd))
	stackSys := stackTop - stackEnd

	stats := GCStats{
		Alloc:      (gcTotalBlocks - gcFreedBlocks) * uint64(bytesPerBlock),
		TotalAlloc: gcTotalAlloc,
		Sys:        uint64(heapEnd - heapStart),
		Mallocs:    gcMallocs,
		Frees:      gcFrees,
		HeapAlloc:  (gcTotalBlocks - gcFreedBlocks) * uint64(bytesPerBlock),
		HeapSys:    heapInuse + heapIdle,
		HeapIdle:   heapIdle,
		HeapInuse:  heapInuse,
		StackInuse: uint64(stackTop - uintptr(getsp())),
		StackSys:   uint64(stackSys),
		GCSys:      uint64(heapEnd - uintptr(metadataStart)),
	}

	unlock(&gcMutex)

	return stats
}
