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

// runtimeContext keeps the G, M, and P for the current 1:1 backend in one
// allocation. Keeping their ownership together makes mexit deterministic while
// leaving the individual objects and links compatible with a later M:N backend.
type runtimeContext struct {
	g g
	m m
	p p

	// root is non-nil for contexts passed through a host-thread API. Such
	// contexts must remain visible to the collector until mexit.
	root unsafe.Pointer
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

// NewProc creates a new G running fn.
//
// The compiler turns a go statement into a call to NewProc. Unlike the old
// lowering, this ABI contains no pthread types: the selected runtime backend
// decides how to provide an M and execute the G.
func NewProc(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr) {
	gp := newproc1(fn, arg, getg())
	if errno := newm(gp.m, stackSize); errno != 0 {
		ctx := gp.context
		releaseG()
		FreeRoot(arg)
		FreeRoot(ctx.root)
		panic("runtime: failed to create new OS thread")
	}
}

// newproc1 creates a runnable G and its initial M/P ownership. The pthread
// backend starts that G immediately; a future scheduler can enqueue the same G
// without changing the compiler ABI.
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
	size := unsafe.Sizeof(runtimeContext{})
	root := AllocRoot(size)
	if root == nil {
		panic("runtime: failed to allocate goroutine context")
	}
	c.Memset(root, 0, size)
	ctx := (*runtimeContext)(root)
	ctx.root = root
	return ctx
}

// newm starts the platform execution resource for mp.
func newm(mp *m, stackSize uintptr) int {
	return newosproc(mp, stackSize)
}

// mstart is the first LLGo runtime function executed on a new M.
func mstart(arg unsafe.Pointer) unsafe.Pointer {
	mp := (*m)(arg)
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mstart context")
		return nil
	}
	gp := mp.curg
	pp := mp.p

	setg(gp)
	casgstatus(gp, _Grunnable, _Grunning)
	setpstatus(pp, _Prunning)

	fn, arg := gp.startfn, gp.startarg
	gp.startfn = nil
	gp.startarg = nil
	ret := fn(arg)
	mexit(mp)
	return ret
}

// mexit tears down the current 1:1 G/M/P context. It does not terminate the
// host thread so both a returning start routine and runtime.Goexit can share
// the same ownership cleanup.
func mexit(mp *m) {
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mexit context")
		return
	}
	gp := mp.curg
	pp := mp.p
	ctx := gp.context
	root := ctx.root
	ownedByLifecycle := currentGUsesLifecycle()
	if !ownedByLifecycle {
		releaseGAndCheckDeadlock()
	}

	casgstatus(gp, _Grunning, _Gdead)
	setpstatus(pp, _Pdead)

	pp.m = nil
	mp.p = nil
	mp.curg = nil
	gp.m = nil

	setg(nil)
	if !ownedByLifecycle && root != nil {
		ctx.root = nil
		FreeRoot(root)
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

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := &ctx.g
	mp := &ctx.m
	pp := &ctx.p

	gp.m = mp
	gp.atomicstatus = status
	gp.goid = nextGoid(gp)
	if callergp != nil {
		gp.parentGoid = callergp.goid
	}
	gp.context = ctx

	mp.curg = gp
	mp.p = pp
	mp.id = nextMid(mp)

	pp.id = nextPid(pp)
	pstatus := uint32(_Pidle)
	if status == _Grunning {
		pstatus = _Prunning
	}
	setpstatus(pp, pstatus)
	pp.m = mp
	retainG()
	return gp
}

// GMPForTesting reports the current runtime ownership graph. It is kept
// internal to the compiler runtime and linked only by LLGo execution tests.
func GMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool) {
	gp := getg()
	if gp == nil || gp.m == nil || gp.m.p == nil {
		return
	}
	mp := gp.m
	pp := mp.p
	ctx := gp.context
	return gp.goid, gp.parentGoid, mp.id, pp.id, readgstatus(gp), readpstatus(pp),
		mp.curg == gp && pp.m == mp && ctx != nil &&
			&ctx.g == gp && &ctx.m == mp && &ctx.p == pp
}

// GStateForTesting reports the packed scheduler state without changing it.
// Execution tests use it to wait until a lifecycle-owned main G has completed
// mexit before allowing the last worker to return.
func GStateForTesting() (count uint64, mainExited bool) {
	return gStateForTesting()
}
