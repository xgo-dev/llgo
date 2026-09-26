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

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
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

	// root is non-nil for contexts passed through a host-thread API. Such
	// contexts must remain visible to the collector until mexit.
	root unsafe.Pointer

	platform runtimeContextPlatform
}

var sched struct {
	goidgen uint64
	midgen  int64
	pidgen  int32

	// gstate packs the live/registered goroutine count with the main-exited
	// bit. The goroutine whose release observes count zero can therefore make
	// the deadlock decision from one atomic result.
	gstate uint64
}

// goroutineStackSize is initialized by the compiler as a read-only constant in
// this package, before any package init or goroutine can run. Zero preserves
// the backend's default. Keep it without a Go initializer so init cannot
// overwrite the configured value. The compiler names this symbol through
// ssa.RuntimeGoroutineStackSizeVar; update both when renaming it.
var goroutineStackSize uintptr

// NewProc creates a new G running fn.
//
// The compiler turns a go statement into a call to NewProc. Unlike the old
// lowering, this ABI contains no host-thread types: the selected runtime backend
// decides how to provide an M and execute the G.
func NewProc(fn goroutineFunc, arg unsafe.Pointer) {
	newprocBackend(fn, arg, goroutineStackSize, getg())
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
	size := runtimeContextAllocSize()
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
	unregisterTraceback(&ctx.g)
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
	registerTraceback(gp, callergp)
	retainG()
	return gp
}

func releaseStartArg(gp *g) {
	if arg := gp.startarg; arg != nil {
		gp.startarg = nil
		FreeRoot(arg)
	}
}

// releaseGAndCheckDeadlock is the sole last-goroutine decision. Main marks its
// exit before releasing its own context, so regardless of release ordering the
// final goroutine observes both facts in the packed atomic state.
func releaseGAndCheckDeadlock() {
	remaining, mainExited := releaseG()
	if remaining == 0 && mainExited {
		fatal("no goroutines (main called runtime.Goexit) - deadlock!")
		c.Exit(2)
	}
}

// Gosched asks the active backend to yield. WebAssembly backends re-queue the
// current G and yield to their scheduler; pthread Gs rely on the host scheduler.
func Gosched() {
	goschedBackend()
}

// GStateForTesting reports the packed scheduler state without changing it.
// Execution tests use it to wait until a lifecycle-owned main G has completed
// mexit before allowing the last worker to return.
func GStateForTesting() (count uint64, mainExited bool) {
	return gState()
}

// NumGoroutine reports the number of live runtime contexts. A go statement
// registers its context before the platform thread is started, matching Go's
// guarantee that the new goroutine is visible when NewProc returns.
func NumGoroutine() int {
	count, _ := gState()
	return int(count)
}
