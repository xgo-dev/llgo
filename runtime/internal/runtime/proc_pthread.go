//go:build !llgo || !wasm || (wasip1 && llgo.wasi_threads)

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

	"github.com/xgo-dev/llgo/runtime/internal/thread"
)

// The pthread backend keeps its one-to-one M/P pair in the G context without
// exposing those fields to other execution-context backends.
type runtimeContextPlatform struct {
	m m
	p p
	// systemG is excluded from the user-goroutine deadlock count.
	systemG bool

	// foreignThreadAttached records that LLGo, rather than the selected GC
	// backend's thread-creation API, attached this host thread. Keep ownership
	// with the native lifecycle because TLS may already be unavailable when an
	// OS thread destructor runs.
	foreignThreadAttached bool

	pool mPoolSlot
}

// Native ABIs agree with Go's runtimeContext layout, so do not add the wasm
// allocator padding to every native goroutine.
func runtimeContextAllocSize() uintptr {
	return unsafe.Sizeof(runtimeContext{})
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	if fn == nil {
		panic("go of nil func value")
	}
	if mPoolStart(fn, arg, stackSize, callergp) {
		return
	}
	gp := newproc1(fn, arg, callergp)
	if errno := newm(gp.m, stackSize); errno != 0 {
		releaseG()
		releaseStartArg(gp)
		freeRuntimeContext(gp.context)
		panic("runtime: failed to create new OS thread")
	}
}

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	mp := &ctx.platform.m
	pp := &ctx.platform.p

	gp.m = mp
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
	return gp
}

func newm(mp *m, stackSize uintptr) int {
	return newosproc(mp, stackSize)
}

func mstart(arg unsafe.Pointer) unsafe.Pointer {
	mp := (*m)(arg)
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mstart context")
		return nil
	}
	registerWasiGCThread()
	for {
		ret, parkCtx := runGoroutine(mp)
		if parkCtx == nil {
			return ret
		}
		mPoolScrubStack()
		if !mPoolPark(parkCtx) {
			finishRetiredM(parkCtx)
			return nil
		}
	}
}

// canRecycleM reports whether mp's OS thread is clean to return to the idle
// worker pool after gp finishes normally.
func canRecycleM(mp *m, gp *g, ctx *runtimeContext) bool {
	// Pool disabled on this backend (e.g. wasm/baremetal).
	if !mPoolEnabled() {
		return false
	}
	// Linux controlled all-threads syscalls require every live OS thread to stay
	// registered in the traceback/allthreads cohort rather than parking detached.
	if allThreadsControlled() {
		return false
	}
	// Main and foreign C threads are owned by the TLS destructor lifecycle.
	if currentGUsesLifecycle() {
		return false
	}
	// Persistent runtime system goroutines exit directly.
	if ctx.platform.systemG {
		return false
	}
	// Threads that initialized thread-local or goroutine-local guards cannot
	// safely reset those guards for a later goroutine.
	if mp.touchedLocal {
		return false
	}
	// Per Go's runtime.LockOSThread contract, exiting while locked must
	// terminate the OS thread rather than reuse it.
	if gp.lockedOSThread != 0 {
		return false
	}
	// Abnormal unwind state must complete teardown via mexit.
	if gp.panic_ != nil || gp.defer_ != nil {
		return false
	}
	return true
}

// runGoroutine runs one goroutine activation on mp in its own non-inlined stack
// frame. Returning before mPoolPark ensures every stack slot and callee-saved
// register touched by the goroutine's entry function is cleared below mstart's
// stack pointer before BDWGC's GC_do_blocking records the idle stack boundary.
//
//go:noinline
func runGoroutine(mp *m) (unsafe.Pointer, *runtimeContext) {
	gp := mp.curg
	pp := mp.p
	if gp == nil || pp == nil {
		fatal("runtime: invalid mstart context")
		return nil, nil
	}

	setg(gp)
	casgstatus(gp, _Grunnable, _Grunning)
	setpstatus(pp, _Prunning)

	fn, fnArg := gp.startfn, gp.startarg
	gp.startfn = nil
	ret := fn(fnArg)

	gp = mp.curg
	pp = mp.p
	if gp == nil || pp == nil {
		fatal("runtime: invalid mexit context")
		return ret, nil
	}
	ctx := gp.context
	if !canRecycleM(mp, gp, ctx) {
		mexit(mp)
		return ret, nil
	}
	releaseStartArg(gp)
	leaveSynctest(gp)
	detachTraceback(gp)
	casgstatus(gp, _Grunning, _Gdead)
	setpstatus(pp, _Pidle)

	savedWaiter := gp.chanWaiter
	*gp = g{chanWaiter: savedWaiter}
	mp.curg = nil
	setg(nil)

	releaseGAndCheckDeadlock()
	return nil, ctx
}

func finishRetiredM(ctx *runtimeContext) {
	mp := &ctx.platform.m
	pp := &ctx.platform.p
	cachedRoots := rootCacheCount(mp)
	clearMediumAllocCache(mp)
	recordRetiredRootCache(mp, cachedRoots)
	mPoolDestroySlot(&ctx.platform.pool)
	setpstatus(pp, _Pdead)
	pp.m = nil
	mp.p = nil
	releaseTracebackThread()
	freeRuntimeContext(ctx)
	unregisterWasiGCThread()
}

func mexit(mp *m) {
	if mp == nil || mp.curg == nil || mp.p == nil {
		fatal("runtime: invalid mexit context")
		return
	}
	gp := mp.curg
	pp := mp.p
	ctx := gp.context
	// The compiler-generated entry wrapper needs the scanned startup record
	// for its whole initial call. Both return and Goexit converge here.
	releaseStartArg(gp)
	clearMediumAllocCache(mp)
	ownedByLifecycle := currentGUsesLifecycle()
	if !ownedByLifecycle && !ctx.platform.systemG {
		releaseGAndCheckDeadlock()
	}

	casgstatus(gp, _Grunning, _Gdead)
	setpstatus(pp, _Pdead)

	pp.m = nil
	mp.p = nil
	mp.curg = nil
	gp.m = nil

	setg(nil)
	if !ownedByLifecycle {
		mPoolDestroySlot(&ctx.platform.pool)
		freeRuntimeContext(ctx)
	}
	if !ownedByLifecycle {
		unregisterWasiGCThread()
	}
}

func goschedBackend() {
	thread.Yield()
}

// MarkSystemGoroutine excludes a persistent runtime service from the user
// goroutine count, so an idle service cannot suppress the main Goexit check.
func MarkSystemGoroutine() {
	gp := getg()
	if gp.context.platform.systemG {
		return
	}
	gp.context.platform.systemG = true
	releaseGAndCheckDeadlock()
}

// MarkUserGoroutine counts a runtime worker while it calls user code. Such a
// callback can still make progress after the main goroutine calls Goexit.
func MarkUserGoroutine() {
	gp := getg()
	if !gp.context.platform.systemG {
		return
	}
	retainG()
	gp.context.platform.systemG = false
}

// SchedulerMultiplexesGoroutinesForTesting reports whether successive
// goroutines may reuse the same M/P pair (true when the native worker pool is enabled).
func SchedulerMultiplexesGoroutinesForTesting() bool { return mPoolEnabled() }

// GMPForTesting reports the current runtime ownership graph.
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
			&ctx.g == gp && &ctx.platform.m == mp && &ctx.platform.p == pp
}
