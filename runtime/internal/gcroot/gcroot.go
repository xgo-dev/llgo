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
	active   *Context
)

// CurrentChain returns the active execution owner's compiler root chain.
func CurrentChain() unsafe.Pointer {
	return currentRootChain
}

// RestoreChain installs a chain captured before a non-local control transfer.
func RestoreChain(chain unsafe.Pointer) {
	currentRootChain = chain
}

// Register adds a suspended context to root enumeration.
func Register(ctx *Context) {
	if ctx == nil || registered(ctx) {
		panic("gcroot: invalid context registration")
	}
	ctx.next = contexts
	contexts = ctx
}

// RegisterActive adds ctx and assigns the existing LLVM root chain to it.
func RegisterActive(ctx *Context) {
	if active != nil {
		panic("gcroot: active context already registered")
	}
	Register(ctx)
	active = ctx
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
	if active == next {
		return
	}
	if active != nil {
		active.chain = currentRootChain
	}
	active = next
	currentRootChain = next.chain
}

// AdoptCurrent marks next active after a target-specific stack switch has
// already restored currentRootChain.
func AdoptCurrent(next *Context) {
	active = next
}

// Unregister removes a suspended context from root enumeration.
func Unregister(ctx *Context) {
	if ctx == nil || ctx == active {
		panic("gcroot: invalid context unregistration")
	}
	link := &contexts
	for *link != nil && *link != ctx {
		link = &(*link).next
	}
	if *link == nil {
		panic("gcroot: context is not registered")
	}
	*link = ctx.next
	ctx.next = nil
	ctx.chain = nil
}

// Visit calls visitor for every root slot in every registered context.
func Visit(visitor func(root *unsafe.Pointer, metadata unsafe.Pointer)) {
	if visitor == nil {
		return
	}
	for ctx := contexts; ctx != nil; ctx = ctx.next {
		chain := ctx.chain
		if ctx == active {
			chain = currentRootChain
		}
		visitChain(chain, visitor)
	}
}

func registered(want *Context) bool {
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
