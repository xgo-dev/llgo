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

// Package wasmresume defines the runtime half of LLGo's experimental
// WebAssembly resumable call ABI.
package wasmresume

import "unsafe"

// SuspendCurrent yields the active resumable frame to its scheduler. The
// compiler replaces calls to SuspendCurrent with a frame-PC transition; no
// function body is linked into the final WebAssembly module.
func SuspendCurrent() {
	panic("wasmresume: SuspendCurrent was not lowered")
}

// Action tells Context what a resume entry did.
type Action uint8

const (
	// Continue means that execution can continue immediately. The resume entry
	// may have pushed a child frame or advanced within the current frame.
	Continue Action = iota

	// Return means that the current frame completed normally.
	Return

	// Suspend returns control to the scheduler without changing the frame chain.
	Suspend
)

// Resume is the non-suspending indirect-call signature for generated entries.
//
//llgo:type C
type Resume func(*Context, *Frame) Action

// Allocator allocates one GC-scanned root block.
//
//llgo:type C
type Allocator func(uintptr) unsafe.Pointer

// Releaser releases one block previously returned by Allocator.
//
//llgo:type C
type Releaser func(unsafe.Pointer)

// Descriptor contains immutable state shared by every invocation of a
// generated function.
type Descriptor struct {
	Resume       Resume
	FrameSize    uintptr
	FrameAlign   uintptr
	UnwindOffset uintptr
	UnwindPC     uint32
}

// Frame is the common prefix of every generated function frame. Generated
// frame types must embed Frame as their first field.
type Frame struct {
	Parent     *Frame
	Descriptor *Descriptor
	PC         uint32
}

// Context owns the active frame chain for one logical goroutine.
type Context struct {
	top      *Frame
	returned *Frame
	storage  frameStorage
}

// Start installs the root frame of a new logical goroutine.
func (c *Context) Start(frame *Frame) {
	if frame == nil || frame.Parent != nil || frame.Descriptor == nil || c.top != nil {
		panic("wasmresume: invalid root frame")
	}
	c.returned = nil
	c.top = frame
}

// AllocateFrame allocates stable, root-scanned storage for a generated frame.
func (c *Context) AllocateFrame(
	size, align uintptr, allocate Allocator,
) unsafe.Pointer {
	return c.storage.allocate(size, align, allocate)
}

// ReleaseFrame reclaims the most recently completed generated frame.
func (c *Context) ReleaseFrame(frame *Frame, release Releaser) {
	if frame == nil || frame.Descriptor == nil {
		panic("wasmresume: invalid completed frame")
	}
	c.storage.releaseFrame(unsafe.Pointer(frame), frame.Descriptor.FrameSize, release)
}

// Unwind discards frames above the defer owner and redirects that owner to its
// generated panic/defer state.
func (c *Context) Unwind(deferFrame unsafe.Pointer, release Releaser) bool {
	if deferFrame == nil {
		return false
	}
	var owner *Frame
	for frame := c.top; frame != nil; frame = frame.Parent {
		descriptor := frame.Descriptor
		if descriptor == nil || descriptor.UnwindOffset == 0 ||
			descriptor.UnwindPC == 0 {
			continue
		}
		slot := (*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(frame), descriptor.UnwindOffset))
		if *slot == deferFrame {
			owner = frame
			break
		}
	}
	if owner == nil {
		return false
	}
	for c.top != owner {
		frame := c.top
		c.top = frame.Parent
		c.storage.releaseFrame(unsafe.Pointer(frame), frame.Descriptor.FrameSize, release)
	}
	c.returned = nil
	owner.PC = owner.Descriptor.UnwindPC
	return true
}

// Close releases every frame storage segment owned by the context.
func (c *Context) Close(release Releaser) {
	c.storage.close(release)
	c.top = nil
	c.returned = nil
}

// Top returns the active frame.
func (c *Context) Top() *Frame {
	return c.top
}

// Push links frame as the active child of the current frame.
func (c *Context) Push(frame *Frame, descriptor *Descriptor) {
	if c.top == nil {
		c.returned = nil
	}
	frame.Parent = c.top
	frame.Descriptor = descriptor
	frame.PC = 0
	c.top = frame
}

// TakeReturned returns the child frame that completed immediately before the
// active frame resumed. It also transfers ownership back to the caller.
func (c *Context) TakeReturned() *Frame {
	frame := c.returned
	c.returned = nil
	return frame
}

// Run resumes the active frame chain until it completes or suspends.
func (c *Context) Run() Action {
	for c.top != nil {
		frame := c.top
		switch frame.Descriptor.Resume(c, frame) {
		case Continue:
		case Return:
			c.top = frame.Parent
			c.returned = frame
		case Suspend:
			return Suspend
		default:
			panic("wasmresume: invalid resume action")
		}
	}
	return Return
}
