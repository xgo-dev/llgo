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

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

// goroutineFunc is the target-independent entry ABI between compiler-generated
// goroutine wrappers and the runtime scheduler.
//
//llgo:type C
type goroutineFunc func(unsafe.Pointer) unsafe.Pointer

// runtimeContext owns one G and its target-specific suspended execution state.
// M and P ownership belongs to the selected scheduler backend and can outlive,
// or be shared by, multiple runtime contexts.
type runtimeContext struct {
	g g

	root     unsafe.Pointer
	platform runtimeContextPlatform
}

var sched struct {
	goidgen uint64
	midgen  int64
	pidgen  int32
}

// NewProc creates a new G running fn.
//
// The compiler turns a go statement into a call to NewProc. Unlike the old
// lowering, this ABI contains no pthread types: the selected runtime backend
// decides how to provide an M and execute the G.
func NewProc(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr) {
	newprocBackend(fn, arg, stackSize, getg())
}

// newproc1 creates target-independent runnable G state. The selected backend
// attaches execution resources and either starts or queues it.
func newproc1(fn goroutineFunc, arg unsafe.Pointer, callergp *g) *g {
	if fn == nil {
		panic("go of nil func value")
	}

	ctx := allocRuntimeContext()
	gp := initRuntimeContext(ctx, callergp, _Grunnable)
	gp.startfn = fn
	gp.startarg = arg
	return gp
}

func allocRuntimeContext() *runtimeContext {
	// LLVM rounds contexts containing 64-bit IDs to this boundary on wasm.
	const contextAlignment = uintptr(unsafe.Sizeof(uint64(0)))
	size := (unsafe.Sizeof(runtimeContext{}) + contextAlignment - 1) &^ (contextAlignment - 1)
	root := AllocRoot(size)
	if root == nil {
		panic("runtime: failed to allocate goroutine context")
	}
	c.Memset(root, 0, size)
	ctx := (*runtimeContext)(root)
	ctx.root = root
	return ctx
}

func freeRuntimeContext(ctx *runtimeContext) {
	if ctx == nil || ctx.root == nil {
		return
	}
	root := ctx.root
	ctx.root = nil
	FreeRoot(root)
}

func initG(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := &ctx.g
	gp.atomicstatus = status
	gp.goid = nextGoid(gp)
	if callergp != nil {
		gp.parentGoid = callergp.goid
	}
	gp.context = ctx
	return gp
}

// Gosched asks the active backend to yield. The WebAssembly fiber backend
// switches to another runnable G; pthread Gs rely on the host thread scheduler.
func Gosched() {
	goschedBackend()
}
