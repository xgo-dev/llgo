//go:build llgo

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package runtime

import "unsafe"

// LocalContext is rooted by the outer Go entry stack frame. The current
// one-thread-per-goroutine backend maps both logical locality kinds to this one
// physical package store.
type LocalContext struct {
	// blocks points at the payload of the most recently allocated local block.
	// The list keeps every block reachable from the outer Go entry stack frame.
	blocks unsafe.Pointer
}

type localBlock struct {
	// next points at the next block's payload, not its header.
	next      unsafe.Pointer
	cacheSlot *uintptr
}

// A callback entered from a native thread can have a G before an entry-local
// context has been installed. Its stack can still be unwound, but generated
// package-local caches cannot be accessed until that context exists.
func callerLocationAvailable() bool { return currentLocalContext != 0 }

// EnterLocalContext installs ctx when the current thread has no local owner.
// A nonzero result means this is a nested Go entry that inherited the returned
// context; in that case ctx is not installed.
func EnterLocalContext(ctx *LocalContext) uintptr {
	previous := currentLocalContext
	if previous == 0 {
		if ctx == nil {
			panic("runtime: nil local context")
		}
		currentLocalContext = uintptr(unsafe.Pointer(ctx))
	}
	return previous
}

// LeaveLocalContext finishes an entry paired with EnterLocalContext. A nested
// entry verifies and retains its inherited context. An outer entry clears ctx
// and releases its package-block roots.
func LeaveLocalContext(ctx *LocalContext, previous uintptr) {
	if previous != 0 {
		if currentLocalContext != previous {
			panic("runtime: local context changed by nested entry")
		}
		return
	}
	if currentLocalContext != uintptr(unsafe.Pointer(ctx)) {
		panic("runtime: leaving inactive local context")
	}
	currentLocalContext = 0
	releaseLocalBlocks(ctx)
}

func leaveCurrentLocalContext() {
	ctx := (*LocalContext)(unsafe.Pointer(currentLocalContext))
	if ctx == nil {
		return
	}
	currentLocalContext = 0
	releaseLocalBlocks(ctx)
}

func releaseLocalBlocks(ctx *LocalContext) {
	data := ctx.blocks
	ctx.blocks = nil
	for data != nil {
		block := localBlockHeader(data)
		next := block.next
		// Do not free block here: an address of a local variable may outlive its
		// owner. Breaking the links lets the GC retain only escaped blocks.
		*block.cacheSlot = 0
		block.next = nil
		block.cacheSlot = nil
		data = next
	}
}

// LocalPackage creates stable, zeroed storage for one generated cache slot in
// the current physical owner. Generated accessors load the slot directly after
// first touch; the block list is retained only as a GC root and teardown list.
//
//go:noinline
func LocalPackage(cacheSlot *uintptr, size, align uintptr) unsafe.Pointer {
	ctx := (*LocalContext)(unsafe.Pointer(currentLocalContext))
	if ctx == nil {
		panic("runtime: local variable accessed outside a Go entry context")
	}
	if cacheSlot == nil {
		panic("runtime: nil local cache slot")
	}
	if data := unsafe.Pointer(*cacheSlot); data != nil {
		return data
	}
	if align == 0 || align&(align-1) != 0 {
		panic("runtime: invalid local package alignment")
	}
	data := newLocalBlock(cacheSlot, size, align)
	localBlockHeader(data).next = ctx.blocks
	ctx.blocks = data
	*cacheSlot = uintptr(data)
	return data
}

func newLocalBlock(cacheSlot *uintptr, size, align uintptr) unsafe.Pointer {
	header := unsafe.Sizeof(localBlock{})
	padding := align - 1
	if size == 0 {
		size = 1
	}
	if header > ^uintptr(0)-padding || header+padding > ^uintptr(0)-size {
		panic("runtime: local package size overflow")
	}
	allocation := AllocZ(header + padding + size)
	if allocation == nil {
		panic("runtime: failed to allocate local package")
	}
	data := unsafe.Pointer((uintptr(allocation) + header + padding) &^ padding)
	block := localBlockHeader(data)
	block.cacheSlot = cacheSlot
	return data
}

func localBlockHeader(data unsafe.Pointer) *localBlock {
	return (*localBlock)(unsafe.Pointer(uintptr(data) - unsafe.Sizeof(localBlock{})))
}
