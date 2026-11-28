//go:build llgo && !wasm && !baremetal

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
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// mPoolSlot holds per-worker parking state. On backends where sig.useFast() is
// true (Darwin macOS 14.4+, Linux, Windows), assignment uses the lock-free
// sig.resetFast / parkIdleFast / readyFast handshake without touching hasWork;
// portable backends synchronize work handoff via hasWork under sig.lock().
type mPoolSlot struct {
	next    *runtimeContext
	sig     chanSignal
	hasWork bool
	pooled  bool
}

var mPool struct {
	mu        chanMutex
	head      *runtimeContext
	idle      int
	maxIdle   int
	idleCount uint32
	inited    uint32
}

func mPoolEnabled() bool { return true }

// MPoolIdleForTesting reports the current number of idle pooled worker threads.
func MPoolIdleForTesting() int { return int(atomic.Load(&mPool.idleCount)) }

func markTouchedLocal() {
	if gp := (*g)(unsafe.Pointer(currentG)); gp != nil && gp.m != nil {
		gp.m.touchedLocal = true
	}
}

// MarkTouchedLocal records that the current OS thread holds goroutine-scoped
// host TLS state (such as a runtime/internal/clite/tls slot) and must exit
// rather than return to the worker pool so native TLS destructors run.
func MarkTouchedLocal() {
	markTouchedLocal()
}

func ensureMPool() {
	if atomic.Load(&mPool.inited) == 2 {
		return
	}
	if _, won := atomic.CompareAndExchange(&mPool.inited, uint32(0), uint32(1)); won {
		mPool.mu.init()
		maxIdle := CPUCount() * 2
		if maxIdle < 8 {
			maxIdle = 8
		}
		if maxIdle > 32 {
			maxIdle = 32
		}
		mPool.maxIdle = maxIdle
		atomic.Store(&mPool.inited, uint32(2))
		return
	}
	for atomic.Load(&mPool.inited) != 2 {
		c.Usleep(1)
	}
}

func mPoolPop() *runtimeContext {
	if atomic.Load(&mPool.idleCount) == 0 {
		return nil
	}
	ensureMPool()
	mPool.mu.Lock()
	ctx := mPool.head
	if ctx != nil {
		mPool.head = ctx.platform.pool.next
		ctx.platform.pool.next = nil
		mPool.idle--
		atomic.Store(&mPool.idleCount, uint32(mPool.idle))
	}
	mPool.mu.Unlock()
	return ctx
}

// mPoolStart assigns a new goroutine to an idle pooled OS thread if available.
// The context keeps its AllocRoot backing, OS thread, BDWGC registration, stack,
// and M/P IDs, while initializing a fresh G with a new goid and traceback node.
func mPoolStart(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) bool {
	if stackSize != goroutineStackSize {
		return false
	}
	ctx := mPoolPop()
	if ctx == nil {
		return false
	}
	if ctx.platform.m.touchedLocal || ctx.platform.m.curg != nil || ctx.g.lockedOSThread != 0 || ctx.g.panic_ != nil || ctx.g.defer_ != nil {
		fatal("runtime: pooled worker retained dirty goroutine state")
	}

	savedWaiter := ctx.g.chanWaiter
	ctx.g = g{chanWaiter: savedWaiter}
	ctx.platform.systemG = false

	gp := &ctx.g
	mp := &ctx.platform.m
	pp := &ctx.platform.p
	gp.atomicstatus = _Grunnable
	gp.goid = nextGoid(gp)
	if callergp != nil {
		gp.parentGoid = callergp.goid
	}
	gp.context = ctx
	gp.m = mp
	gp.startfn = fn
	gp.startarg = arg
	mp.curg = gp
	mp.p = pp
	pp.m = mp
	setpstatus(pp, _Pidle)

	inheritSynctest(gp, callergp)
	registerTraceback(gp, callergp)
	retainG()

	slot := &ctx.platform.pool
	if slot.sig.useFast() {
		slot.sig.readyFast()
		return true
	}
	slot.sig.lock()
	slot.hasWork = true
	slot.sig.ready()
	slot.sig.unlock()
	return true
}

func mPoolPushIdle(ctx *runtimeContext) bool {
	mPool.mu.Lock()
	if mPool.idle >= mPool.maxIdle {
		mPool.mu.Unlock()
		return false
	}
	ctx.platform.pool.next = mPool.head
	mPool.head = ctx
	mPool.idle++
	atomic.Store(&mPool.idleCount, uint32(mPool.idle))
	mPool.mu.Unlock()
	return true
}

// mPoolPark parks an idle worker thread in the pool until another goroutine is
// assigned. It returns false when the pool is full so the caller can exit the
// OS thread cleanly.
func mPoolPark(ctx *runtimeContext) bool {
	ensureMPool()
	slot := &ctx.platform.pool
	if !slot.pooled {
		slot.sig.initFast()
		slot.pooled = true
	}
	if slot.sig.useFast() {
		slot.sig.resetFast()
		if !mPoolPushIdle(ctx) {
			return false
		}
		slot.sig.parkIdleFast()
		return true
	}
	slot.sig.lock()
	slot.hasWork = false
	if !mPoolPushIdle(ctx) {
		slot.sig.unlock()
		return false
	}

	for !slot.hasWork {
		slot.sig.parkIdle()
	}
	slot.hasWork = false
	slot.sig.unlock()
	return true
}

//go:linkname llgoScrubStack C.llgo_scrub_stack
func llgoScrubStack()

func mPoolScrubStack() {
	llgoScrubStack()
}

func mPoolDestroySlot(slot *mPoolSlot) {
	if slot.pooled {
		slot.sig.destroy()
		slot.pooled = false
	}
}
