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

// Package gcroot owns LLGo's per-G compiler root chains.
package gcroot

import "unsafe"

// Context stores one suspended execution owner's compiler root chain.
type Context struct {
	next  *Context
	chain unsafe.Pointer
}

type frameMap struct {
	numRoots uint32
	numMeta  uint32
}

type stackEntry struct {
	next *stackEntry
	m    *frameMap
}

var (
	contexts *Context
)

// CurrentChain returns the active execution owner's compiler root chain.
func CurrentChain() unsafe.Pointer {
	return unsafe.Pointer(currentRootChain)
}

// RestoreChain installs a chain captured before a non-local control transfer.
func RestoreChain(chain unsafe.Pointer) {
	currentRootChain = uintptr(chain)
}

// BeginSJLJReplay marks the non-local return path before siglongjmp. The wasm
// SJLJ/Asyncify lowering replays function entries while it searches for the
// destination setjmp. Compiler root frames on that discarded path must not be
// linked into the destination's saved chain.
//
// This function runs at the longjmp boundary. Keep it free of calls and
// allocations so it cannot acquire a compiler-maintained root frame itself.
func BeginSJLJReplay() {
	sjljReplaying = true
	// Prevent collection or a context switch while the restored stack and its
	// compiler root chain temporarily describe different call paths.
	rebuilding = true
}

// FinishSJLJReplay marks the destination setjmp's root chain usable again.
func FinishSJLJReplay() {
	if !sjljReplaying {
		return
	}
	sjljReplaying = false
	rebuilding = false
}

// Register adds a suspended context to root enumeration.
func Register(ctx *Context) {
	lockRegistry()
	if ctx == nil || registeredLocked(ctx) {
		unlockRegistry()
		panic("gcroot: invalid context registration")
	}
	ctx.next = contexts
	contexts = ctx
	unlockRegistry()
}

// RegisterActive adds ctx and marks it active. Visit reads its chain directly
// from currentRootChain until Switch or BeginRebuild saves it in the context.
func RegisterActive(ctx *Context) {
	if activeContext != 0 {
		panic("gcroot: active context already registered")
	}
	Register(ctx)
	activeContext = uintptr(unsafe.Pointer(ctx))
}

// Switch saves the active chain and installs next's chain.
func Switch(next *Context) {
	if next == nil {
		panic("gcroot: switch to nil context")
	}
	SwitchAtBoundary(next)
}

// SwitchAtBoundary saves the active chain and installs next's chain.
//
// This function is called between a context wrapper's root-frame setup and
// the target-specific stack switch. Keep it free of calls and allocations so
// it cannot acquire a compiler-maintained root frame of its own.
func SwitchAtBoundary(next *Context) {
	active := (*Context)(unsafe.Pointer(activeContext))
	if active == next {
		return
	}
	if active != nil {
		active.chain = unsafe.Pointer(currentRootChain)
	}
	activeContext = uintptr(unsafe.Pointer(next))
	if next == nil {
		currentRootChain = 0
	} else {
		currentRootChain = uintptr(next.chain)
	}
}

// BeginRebuild switches ownership to next but discards its saved stack-root
// links. Asyncify restores the stack by replaying function entries, which
// rebuilds those links at the same frame addresses. Reinstalling the saved
// links before that replay would link each frame to itself.
//
// This function runs at a stack-switch boundary. Keep it free of calls and
// allocations so it cannot acquire a compiler-maintained root frame itself.
func BeginRebuild(next *Context) {
	active := (*Context)(unsafe.Pointer(activeContext))
	if active == next {
		return
	}
	if active != nil {
		active.chain = unsafe.Pointer(currentRootChain)
	}
	activeContext = uintptr(unsafe.Pointer(next))
	next.chain = nil
	currentRootChain = 0
	rebuilding = true
}

// FinishRebuild marks an Asyncify stack replay complete.
func FinishRebuild() {
	rebuilding = false
}

// Rebuilding reports whether Asyncify is replaying function entries.
func Rebuilding() bool {
	return rebuilding
}

// AdoptCurrent marks next active after a target-specific stack switch has
// already restored currentRootChain.
func AdoptCurrent(next *Context) {
	activeContext = uintptr(unsafe.Pointer(next))
}

// SuspendCurrent detaches ctx from a native stack that is about to be
// discarded by a host event-loop unwind. A later callback entry builds a new
// compiler root chain and adopts ctx again.
//
// This function runs at an unwind boundary. Keep its valid path free of
// allocations so it cannot retain the stack it is detaching.
func SuspendCurrent(ctx uintptr) {
	if ctx == 0 || ctx != activeContext {
		panicSuspendInactiveContext()
	}
	(*Context)(unsafe.Pointer(ctx)).chain = nil
	activeContext = 0
	currentRootChain = 0
}

//go:noinline
func panicSuspendInactiveContext() {
	panic("gcroot: suspending inactive context")
}

// PublishCurrent saves this worker's root chain before it acknowledges a
// stop-the-world request.
func PublishCurrent() {
	active := (*Context)(unsafe.Pointer(activeContext))
	if active != nil {
		active.chain = unsafe.Pointer(currentRootChain)
	}
}

// Unregister removes a suspended context from root enumeration.
func Unregister(ctx *Context) {
	if ctx == nil || uintptr(unsafe.Pointer(ctx)) == activeContext {
		panic("gcroot: invalid context unregistration")
	}
	lockRegistry()
	link := &contexts
	for *link != nil && *link != ctx {
		link = &(*link).next
	}
	if *link == nil {
		unlockRegistry()
		panic("gcroot: context is not registered")
	}
	*link = ctx.next
	ctx.next = nil
	ctx.chain = nil
	unlockRegistry()
}

// Visit calls visitor for every root slot in every registered context.
func Visit(visitor func(root *unsafe.Pointer, metadata unsafe.Pointer)) {
	if visitor == nil {
		return
	}
	lockRegistry()
	active := (*Context)(unsafe.Pointer(activeContext))
	for ctx := contexts; ctx != nil; ctx = ctx.next {
		chain := ctx.chain
		if ctx == active {
			chain = unsafe.Pointer(currentRootChain)
		}
		visitChain(chain, visitor)
	}
	unlockRegistry()
}

func registeredLocked(want *Context) bool {
	for ctx := contexts; ctx != nil; ctx = ctx.next {
		if ctx == want {
			return true
		}
	}
	return false
}

func visitChain(chain unsafe.Pointer, visitor func(*unsafe.Pointer, unsafe.Pointer)) {
	const pointerSize = unsafe.Sizeof(uintptr(0))
	for entry := (*stackEntry)(chain); entry != nil; entry = entry.next {
		if entry.m == nil || entry.m.numMeta > entry.m.numRoots {
			panic("gcroot: invalid compiler root frame")
		}
		roots := unsafe.Add(unsafe.Pointer(entry), unsafe.Sizeof(stackEntry{}))
		metadata := unsafe.Add(unsafe.Pointer(entry.m), unsafe.Sizeof(frameMap{}))
		for i := uint32(0); i < entry.m.numRoots; i++ {
			var meta unsafe.Pointer
			if i < entry.m.numMeta {
				meta = *(*unsafe.Pointer)(unsafe.Add(metadata, uintptr(i)*pointerSize))
			}
			root := (*unsafe.Pointer)(unsafe.Add(roots, uintptr(i)*pointerSize))
			visitor(root, meta)
		}
	}
}
