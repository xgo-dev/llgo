//go:build baremetal

/*
 * Copyright (c) 2018-2025 The TinyGo Authors. All rights reserved.
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package tinygogc implements a conservative mark-and-sweep garbage collector
// for baremetal environments where the standard Go runtime and bdwgc are unavailable.
//
// This implementation is based on TinyGo's GC and is designed for resource-constrained
// embedded systems. It uses a block-based allocator with conservative pointer scanning.
//
// Build tags:
//   - baremetal: Enables this GC for baremetal targets
//
// Memory Layout:
// The heap is divided into fixed-size blocks (32 bytes on 64-bit). Metadata is stored
// at the end of the heap, using 2 bits per block to track state (free/head/tail/mark).
package tinygogc

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

const gcDebug = false

// blockState stores the four states in which a block can be. It is two bits in
// size.
const (
	blockStateFree uint8 = 0 // 00
	blockStateHead uint8 = 1 // 01
	blockStateTail uint8 = 2 // 10
	blockStateMark uint8 = 3 // 11
	blockStateMask uint8 = 3 // 11
)

// The byte value of a block where every block is a 'tail' block.
const blockStateByteAllTails = 0 |
	uint8(blockStateTail<<(stateBits*3)) |
	uint8(blockStateTail<<(stateBits*2)) |
	uint8(blockStateTail<<(stateBits*1)) |
	uint8(blockStateTail<<(stateBits*0))

var (
	heapStart     uintptr        // start address of heap area
	heapEnd       uintptr        // end address of heap area
	globalsStart  uintptr        // start address of global variable area
	globalsEnd    uintptr        // end address of global variable area
	stackTop      uintptr        // the top of stack
	endBlock      uintptr        // GC end block index
	metadataStart unsafe.Pointer // start address of GC metadata

	nextAlloc     uintptr // the next block that should be tried by the allocator
	gcTotalAlloc  uint64  // total number of bytes allocated
	gcTotalBlocks uint64  // total number of allocated blocks
	gcMallocs     uint64  // total number of allocations
	gcFrees       uint64  // total number of objects freed
	gcFreedBlocks uint64  // total number of freed blocks

	// stackOverflow is a flag which is set when the GC scans too deep while marking.
	// After it is set, all marked allocations must be re-scanned.
	markStackOverflow bool

	// zeroSizedAlloc is just a sentinel that gets returned when allocating 0 bytes.
	zeroSizedAlloc uint8

	gcMutex  mutex // gcMutex protects GC related variables
	isGCInit bool  // isGCInit indicates GC initialization state
)

// Some globals + constants for the entire GC.

const (
	wordsPerBlock      = 4 // number of pointers in an allocated block
	bytesPerBlock      = wordsPerBlock * unsafe.Sizeof(heapStart)
	stateBits          = 2 // how many bits a block state takes (see blockState type)
	blocksPerStateByte = 8 / stateBits
	markStackSize      = 8 * unsafe.Sizeof((*int)(nil)) // number of to-be-marked blocks to queue before forcing a rescan
)

// this function MUST be initalized first, which means it's required to be initalized before runtime
func initGC() {
	// reserve 2K blocks for libc internal malloc, we cannot wrap those internal functions
	heapStart = uintptr(unsafe.Pointer(&_heapStart)) + 2048
	heapEnd = uintptr(unsafe.Pointer(&_heapEnd))
	globalsStart = uintptr(unsafe.Pointer(&_globals_start))
	globalsEnd = uintptr(unsafe.Pointer(&_globals_end))
	totalSize := heapEnd - heapStart
	metadataSize := (totalSize + blocksPerStateByte*bytesPerBlock) / (1 + blocksPerStateByte*bytesPerBlock)
	metadataStart = unsafe.Pointer(heapEnd - metadataSize)
	endBlock = (uintptr(metadataStart) - heapStart) / bytesPerBlock
	stackTop = uintptr(unsafe.Pointer(&_stackStart))

	c.Memset(metadataStart, 0, metadataSize)
}

func lazyInit() {
	if !isGCInit {
		initGC()
		isGCInit = true
	}
}

func gcPanic(s *c.Char) {
	c.Printf(c.Str("%s"), s)
	c.Exit(2)
}

// blockFromAddr returns a block given an address somewhere in the heap (which
// might not be heap-aligned).
func blockFromAddr(addr uintptr) uintptr {
	if addr < heapStart || addr >= uintptr(metadataStart) {
		gcPanic(c.Str("gc: trying to get block from invalid address"))
	}
	return (addr - heapStart) / bytesPerBlock
}

// Return a pointer to the start of the allocated object.
func gcPointerOf(blockAddr uintptr) unsafe.Pointer {
	return unsafe.Pointer(gcAddressOf(blockAddr))
}

// Return the address of the start of the allocated object.
func gcAddressOf(blockAddr uintptr) uintptr {
	addr := heapStart + blockAddr*bytesPerBlock
	if addr > uintptr(metadataStart) {
		gcPanic(c.Str("gc: block pointing inside metadata"))
	}
	return addr
}

// findHead returns the head (first block) of an object, assuming the block
// points to an allocated object. It returns the same block if this block
// already points to the head.
func gcFindHead(blockAddr uintptr) uintptr {
	for {
		// Optimization: check whether the current block state byte (which
		// contains the state of multiple blocks) is composed entirely of tail
		// blocks. If so, we can skip back to the last block in the previous
		// state byte.
		// This optimization speeds up findHead for pointers that point into a
		// large allocation.
		stateByte := gcStateByteOf(blockAddr)
		if stateByte == blockStateByteAllTails {
			blockAddr -= (blockAddr % blocksPerStateByte) + 1
			continue
		}

		// Check whether we've found a non-tail block, which means we found the
		// head.
		state := gcStateFromByte(blockAddr, stateByte)
		if state != blockStateTail {
			break
		}
		blockAddr--
	}
	if gcStateOf(blockAddr) != blockStateHead && gcStateOf(blockAddr) != blockStateMark {
		gcPanic(c.Str("gc: found tail without head"))
	}
	return blockAddr
}

// findNext returns the first block just past the end of the tail. This may or
// may not be the head of an object.
func gcFindNext(blockAddr uintptr) uintptr {
	if gcStateOf(blockAddr) == blockStateHead || gcStateOf(blockAddr) == blockStateMark {
		blockAddr++
	}
	for gcAddressOf(blockAddr) < uintptr(metadataStart) && gcStateOf(blockAddr) == blockStateTail {
		blockAddr++
	}
	return blockAddr
}

func gcStateByteOf(blockAddr uintptr) byte {
	return *(*uint8)(unsafe.Add(metadataStart, blockAddr/blocksPerStateByte))
}

// Return the block state given a state byte. The state byte must have been
// obtained using b.stateByte(), otherwise the result is incorrect.
func gcStateFromByte(blockAddr uintptr, stateByte byte) uint8 {
	return uint8(stateByte>>((blockAddr%blocksPerStateByte)*stateBits)) & blockStateMask
}

// State returns the current block state.
func gcStateOf(blockAddr uintptr) uint8 {
	return gcStateFromByte(blockAddr, gcStateByteOf(blockAddr))
}

// setState sets the current block to the given state, which must contain more
// bits than the current state. Allowed transitions: from free to any state and
// from head to mark.
func gcSetState(blockAddr uintptr, newState uint8) {
	stateBytePtr := (*uint8)(unsafe.Add(metadataStart, blockAddr/blocksPerStateByte))
	*stateBytePtr |= uint8(newState << ((blockAddr % blocksPerStateByte) * stateBits))
	if gcStateOf(blockAddr) != newState {
		gcPanic(c.Str("gc: setState() was not successful"))
	}
}

// markFree sets the block state to free, no matter what state it was in before.
func gcMarkFree(blockAddr uintptr) {
	stateBytePtr := (*uint8)(unsafe.Add(metadataStart, blockAddr/blocksPerStateByte))
	*stateBytePtr &^= uint8(blockStateMask << ((blockAddr % blocksPerStateByte) * stateBits))
	if gcStateOf(blockAddr) != blockStateFree {
		gcPanic(c.Str("gc: markFree() was not successful"))
	}
	*(*[wordsPerBlock]uintptr)(unsafe.Pointer(gcAddressOf(blockAddr))) = [wordsPerBlock]uintptr{}
}

// unmark changes the state of the block from mark to head. It must be marked
// before calling this function.
func gcUnmark(blockAddr uintptr) {
	if gcStateOf(blockAddr) != blockStateMark {
		gcPanic(c.Str("gc: unmark() on a block that is not marked"))
	}
	clearMask := blockStateMask ^ blockStateHead // the bits to clear from the state
	stateBytePtr := (*uint8)(unsafe.Add(metadataStart, blockAddr/blocksPerStateByte))
	*stateBytePtr &^= uint8(clearMask << ((blockAddr % blocksPerStateByte) * stateBits))
	if gcStateOf(blockAddr) != blockStateHead {
		gcPanic(c.Str("gc: unmark() was not successful"))
	}
}

func isOnHeap(ptr uintptr) bool {
	return ptr >= heapStart && ptr < uintptr(metadataStart)
}

func isPointer(ptr uintptr) bool {
	// TODO: implement precise GC
	return isOnHeap(ptr)
}

// alloc tries to find some free space on the heap, possibly doing a garbage
// collection cycle if needed. If no space is free, it panics.
//
//go:noinline
func Alloc(size uintptr) unsafe.Pointer {
	if size == 0 {
		return unsafe.Pointer(&zeroSizedAlloc)
	}
	lock(&gcMutex)
	lazyInit()

	gcTotalAlloc += uint64(size)
	gcMallocs++

	neededBlocks := (size + (bytesPerBlock - 1)) / bytesPerBlock
	gcTotalBlocks += uint64(neededBlocks)

	// Continue looping until a run of free blocks has been found that fits the
	// requested size.
	index := nextAlloc
	numFreeBlocks := uintptr(0)
	heapScanCount := uint8(0)
	for {
		if index == nextAlloc {
			if heapScanCount == 0 {
				heapScanCount = 1
			} else if heapScanCount == 1 {
				// The entire heap has been searched for free memory, but none
				// could be found. Run a garbage collection cycle to reclaim
				// free memory and try again.
				heapScanCount = 2
				freeBytes := gc()
				heapSize := uintptr(metadataStart) - heapStart
				if freeBytes < heapSize/3 {
					// Ensure there is at least 33% headroom.
					// This percentage was arbitrarily chosen, and may need to
					// be tuned in the future.
					growHeap()
				}
			} else {
				// Even after garbage collection, no free memory could be found.
				// Try to increase heap size.
				if growHeap() {
					// Success, the heap was increased in size. Try again with a
					// larger heap.
				} else {
					// Unfortunately the heap could not be increased. This
					// happens on baremetal systems for example (where all
					// available RAM has already been dedicated to the heap).
					gcPanic(c.Str("out of memory"))
				}
			}
		}

		// Wrap around the end of the heap.
		if index == endBlock {
			index = 0
			// Reset numFreeBlocks as allocations cannot wrap.
			numFreeBlocks = 0
			// In rare cases, the initial heap might be so small that there are
			// no blocks at all. In this case, it's better to jump back to the
			// start of the loop and try again, until the GC realizes there is
			// no memory and grows the heap.
			// This can sometimes happen on WebAssembly, where the initial heap
			// is created by whatever is left on the last memory page.
			continue
		}

		// Is the block we're looking at free?
		if gcStateOf(index) != blockStateFree {
			// This block is in use. Try again from this point.
			numFreeBlocks = 0
			index++
			continue
		}
		numFreeBlocks++
		index++

		// Are we finished?
		if numFreeBlocks == neededBlocks {
			// Found a big enough range of free blocks!
			nextAlloc = index
			thisAlloc := index - neededBlocks

			// Set the following blocks as being allocated.
			gcSetState(thisAlloc, blockStateHead)
			for i := thisAlloc + 1; i != nextAlloc; i++ {
				gcSetState(i, blockStateTail)
			}
			unlock(&gcMutex)
			// Return a pointer to this allocation.
			return c.Memset(gcPointerOf(thisAlloc), 0, size)
		}
	}
}

func Realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	if ptr == nil {
		return Alloc(size)
	}
	lock(&gcMutex)
	lazyInit()
	unlock(&gcMutex)

	ptrAddress := uintptr(ptr)
	endOfTailAddress := gcAddressOf(gcFindNext(blockFromAddr(ptrAddress)))

	// this might be a few bytes longer than the original size of
	// ptr, because we align to full blocks of size bytesPerBlock
	oldSize := endOfTailAddress - ptrAddress
	if size <= oldSize {
		return ptr
	}

	newAlloc := Alloc(size)
	c.Memcpy(newAlloc, ptr, oldSize)
	free(ptr)

	return newAlloc
}

func free(ptr unsafe.Pointer) {
	// TODO: free blocks on request, when the compiler knows they're unused.
}

func GC() uintptr {
	lock(&gcMutex)
	freeBytes := gc()
	unlock(&gcMutex)
	return freeBytes
}

// runGC performs a garbage collection cycle. It is the internal implementation
// of the runtime.GC() function. The difference is that it returns the number of
// free bytes in the heap after the GC is finished.
func gc() (freeBytes uintptr) {
	lazyInit()

	if gcDebug {
		println("running collection cycle...")
	}

	// Mark phase: mark all reachable objects, recursively.
	gcMarkReachable()

	finishMark()

	// If we're using threads, resume all other threads before starting the
	// sweep.
	gcResumeWorld()

	// Sweep phase: free all non-marked objects and unmark marked objects for
	// the next collection cycle.
	freeBytes = sweep()

	return
}

// markRoots reads all pointers from start to end (exclusive) and if they look
// like a heap pointer and are unmarked, marks them and scans that object as
// well (recursively). The start and end parameters must be valid pointers and
// must be aligned.
func markRoots(start, end uintptr) {
	if start >= end {
		gcPanic(c.Str("gc: unexpected range to mark"))
	}
	// Reduce the end bound to avoid reading too far on platforms where pointer alignment is smaller than pointer size.
	// If the size of the range is 0, then end will be slightly below start after this.
	end -= unsafe.Sizeof(end) - unsafe.Alignof(end)

	for addr := start; addr < end; addr += unsafe.Alignof(addr) {
		root := *(*uintptr)(unsafe.Pointer(addr))
		markRoot(addr, root)
	}
}

// startMark starts the marking process on a root and all of its children.
func startMark(root uintptr) {
	var stack [markStackSize]uintptr
	stack[0] = root
	gcSetState(root, blockStateMark)
	stackLen := 1
	for stackLen > 0 {
		// Pop a block off of the stack.
		stackLen--
		block := stack[stackLen]

		start, end := gcAddressOf(block), gcAddressOf(gcFindNext(block))

		for addr := start; addr != end; addr += unsafe.Alignof(addr) {
			// Load the word.
			word := *(*uintptr)(unsafe.Pointer(addr))

			if !isPointer(word) {
				// Not a heap pointer.
				continue
			}

			// Find the corresponding memory block.
			referencedBlock := blockFromAddr(word)

			if gcStateOf(referencedBlock) == blockStateFree {
				// The to-be-marked object doesn't actually exist.
				// This is probably a false positive.
				continue
			}

			// Move to the block's head.
			referencedBlock = gcFindHead(referencedBlock)

			if gcStateOf(referencedBlock) == blockStateMark {
				// The block has already been marked by something else.
				continue
			}

			// Mark block.
			gcSetState(referencedBlock, blockStateMark)

			if stackLen == len(stack) {
				// The stack is full.
				// It is necessary to rescan all marked blocks once we are done.
				markStackOverflow = true
				if gcDebug {
					println("gc stack overflowed")
				}
				continue
			}

			// Push the pointer onto the stack to be scanned later.
			stack[stackLen] = referencedBlock
			stackLen++
		}
	}
}

// finishMark finishes the marking process by processing all stack overflows.
func finishMark() {
	for markStackOverflow {
		// Re-mark all blocks.
		markStackOverflow = false
		for block := uintptr(0); block < endBlock; block++ {
			if gcStateOf(block) != blockStateMark {
				// Block is not marked, so we do not need to rescan it.
				continue
			}

			// Re-mark the block.
			startMark(block)
		}
	}
}

// mark a GC root at the address addr.
func markRoot(addr, root uintptr) {
	if isOnHeap(root) {
		block := blockFromAddr(root)
		if gcStateOf(block) == blockStateFree {
			// The to-be-marked object doesn't actually exist.
			// This could either be a dangling pointer (oops!) but most likely
			// just a false positive.
			return
		}
		head := gcFindHead(block)

		if gcStateOf(head) != blockStateMark {
			startMark(head)
		}
	}
}

// Sweep goes through all memory and frees unmarked
// It returns how many bytes are free in the heap after the sweep.
func sweep() (freeBytes uintptr) {
	freeCurrentObject := false
	var freed uint64

	for block := uintptr(0); block < endBlock; block++ {
		switch gcStateOf(block) {
		case blockStateHead:
			// Unmarked head. Free it, including all tail blocks following it.
			gcMarkFree(block)
			freeCurrentObject = true
			gcFrees++
			freed++
		case blockStateTail:
			if freeCurrentObject {
				// This is a tail object following an unmarked head.
				// Free it now.
				gcMarkFree(block)
				freed++
			}
		case blockStateMark:
			// This is a marked object. The next tail blocks must not be freed,
			// but the mark bit must be removed so the next GC cycle will
			// collect this object if it is unreferenced then.
			gcUnmark(block)
			freeCurrentObject = false
		case blockStateFree:
			freeBytes += bytesPerBlock
		}
	}
	gcFreedBlocks += freed
	freeBytes += uintptr(freed) * bytesPerBlock
	return
}

// growHeap tries to grow the heap size. It returns true if it succeeds, false
// otherwise.
func growHeap() bool {
	// On baremetal, there is no way the heap can be grown.
	return false
}

func gcMarkReachable() {
	markRoots(uintptr(getsp()), stackTop)
	markRoots(globalsStart, globalsEnd)
}

func gcResumeWorld() {
	// Nothing to do here (single threaded).
}
