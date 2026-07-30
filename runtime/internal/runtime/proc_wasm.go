//go:build llgo && js && wasm && !llgo.wasm_resume

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

	"github.com/goplus/llgo/runtime/internal/runqueue"
	"github.com/goplus/llgo/runtime/internal/wasmcontext"
)

type runtimeContextPlatform struct {
	context    wasmcontext.Context
	runqNext   *g
	runqQueued bool
}

var wasmSched struct {
	m       m
	p       p
	runq    runqueue.Queue[*g]
	retired *runtimeContext
	started bool
}

func initRuntimeContext(ctx *runtimeContext, callergp *g, status uint32) *g {
	gp := initG(ctx, callergp, status)
	if status == _Grunning {
		initWasmScheduler(gp)
	}
	return gp
}

func initWasmScheduler(gp *g) {
	if wasmSched.started {
		fatal("runtime: WebAssembly scheduler initialized twice")
		return
	}
	wasmSched.started = true
	mp := &wasmSched.m
	pp := &wasmSched.p
	mp.curg = gp
	mp.p = pp
	mp.id = nextMid(mp)
	pp.id = nextPid(pp)
	setpstatus(pp, _Prunning)
	pp.m = mp
	gp.m = mp
}

func newprocBackend(fn goroutineFunc, arg unsafe.Pointer, stackSize uintptr, callergp *g) {
	gp := newproc1(fn, arg, callergp)
	initWasmFiber(gp, stackSize)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
	}
}

func initWasmFiber(gp *g, stackSize uintptr) {
	platform := &gp.context.platform
	if !platform.context.Init(
		wasmcontext.Entry(wasmGStart),
		unsafe.Pointer(gp),
		stackSize,
		AllocRoot,
		FreeRoot,
	) {
		panic("runtime: failed to allocate WebAssembly goroutine stack")
	}
}

func ensureCurrentWasmFiber(gp *g) {
	context := &gp.context.platform.context
	if context.Ready() {
		return
	}
	if !context.InitCurrent(AllocRoot) {
		panic("runtime: failed to allocate WebAssembly goroutine stack")
	}
}

func wasmGStart(arg unsafe.Pointer) {
	gp := (*g)(arg)
	if gp == nil || getg() != gp {
		fatal("runtime: invalid WebAssembly goroutine entry")
		return
	}
	reapRetiredWasmG()
	fn, fnarg := gp.startfn, gp.startarg
	gp.startfn = nil
	gp.startarg = nil
	fn(fnarg)
	goexitBackend(gp)
}

func goschedBackend() {
	gp := getg()
	casgstatus(gp, _Grunning, _Grunnable)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
		return
	}
	next := popWasmRunq()
	if next == gp {
		casgstatus(gp, _Grunnable, _Grunning)
		return
	}
	resumeWasmG(gp, next)
}

func gopark() {
	gp := getg()
	casgstatus(gp, _Grunning, _Gwaiting)
	next := popWasmRunq()
	if next == nil {
		fatal("all goroutines are asleep - deadlock!")
		return
	}
	resumeWasmG(gp, next)
}

func goready(gp *g) {
	if gp == nil {
		fatal("runtime: ready of nil goroutine")
		return
	}
	casgstatus(gp, _Gwaiting, _Grunnable)
	if !wasmSched.runq.Push(gp) {
		fatal("runtime: invalid run queue insertion")
	}
}

func resumeWasmG(old, next *g) {
	if old == nil || next == nil || next.context == nil {
		fatal("runtime: invalid WebAssembly context switch")
		return
	}
	ensureCurrentWasmFiber(old)
	if !next.context.platform.context.Ready() {
		fatal("runtime: uninitialized WebAssembly goroutine context")
		return
	}

	casgstatus(next, _Grunnable, _Grunning)
	mp := &wasmSched.m
	old.m = nil
	next.m = mp
	mp.curg = next
	setg(next)
	old.context.platform.context.Swap(&next.context.platform.context)
	reapRetiredWasmG()
}

func goexitBackend(gp *g) {
	casgstatus(gp, _Grunning, _Gdead)
	if wasmSched.retired != nil {
		fatal("runtime: unreaped WebAssembly goroutine")
		return
	}

	next := popWasmRunq()
	if next == nil {
		if gp.isMain {
			fatal("no goroutines (main called runtime.Goexit) - deadlock!")
		} else {
			fatal("all goroutines are asleep - deadlock!")
		}
		return
	}

	wasmSched.retired = gp.context
	resumeDeadWasmG(gp, next)
}

func resumeDeadWasmG(old, next *g) {
	ensureCurrentWasmFiber(old)
	casgstatus(next, _Grunnable, _Grunning)
	mp := &wasmSched.m
	old.m = nil
	next.m = mp
	mp.curg = next
	setg(next)
	old.context.platform.context.Swap(&next.context.platform.context)
	fatal("runtime: resumed dead WebAssembly goroutine")
}

func reapRetiredWasmG() {
	ctx := wasmSched.retired
	if ctx == nil {
		return
	}
	wasmSched.retired = nil
	ctx.platform.context.Close(FreeRoot)
	freeRuntimeContext(ctx)
}

func popWasmRunq() *g {
	return wasmSched.runq.Pop()
}

// CurrentGForTesting returns an opaque handle suitable for ReadyForTesting.
func CurrentGForTesting() unsafe.Pointer {
	return unsafe.Pointer(getg())
}

// ParkForTesting parks the current G until another G marks it runnable.
func ParkForTesting() {
	gopark()
}

// ReadyForTesting makes a G previously parked by ParkForTesting runnable.
func ReadyForTesting(handle unsafe.Pointer) {
	goready((*g)(handle))
}

// SchedulerStateForTesting reports single-worker queue and ownership state.
func SchedulerStateForTesting() (runq uintptr, mid int64, pid int32) {
	return wasmSched.runq.Len(), wasmSched.m.id, wasmSched.p.id
}

func GMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool) {
	gp := getg()
	if gp == nil || gp.m == nil || gp.m.p == nil {
		return
	}
	mp := gp.m
	pp := mp.p
	ctx := gp.context
	return gp.goid, gp.parentGoid, mp.id, pp.id, readgstatus(gp), readpstatus(pp),
		mp == &wasmSched.m && pp == &wasmSched.p &&
			mp.curg == gp && pp.m == mp && ctx != nil && &ctx.g == gp
}
